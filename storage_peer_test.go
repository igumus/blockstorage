package blockstorage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

const notExistsCid = "bafkreicbhkvymvquwrtgsxbed6imq5ec6526it55c3kp5lxpcjujyg7a4m"
const bootstrapListenAddrString = "/ip4/127.0.0.1/tcp/3001"
const bootstrapPeerFormat = bootstrapListenAddrString + "/p2p/%s"

const basePeerPort = 5000
const peerListenAddrStringFormat = "/ip4/127.0.0.1/tcp/%d"

func generateKeyPair(ctx context.Context) (crypto.PrivKey, error) {

	sk, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}
	return sk, nil
}

func convertPeers(peers []string) []peer.AddrInfo {
	pinfos := make([]peer.AddrInfo, len(peers))
	for i, addr := range peers {
		maddr := ma.StringCast(addr)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatalln(err)
		}
		pinfos[i] = *p
	}
	return pinfos
}

func makeHost(ctx context.Context, listenAddr string) (host.Host, error) {
	sk, err := generateKeyPair(ctx)
	if err != nil {
		log.Printf("err: generation key pair failed: %s\n", err.Error())
		return nil, err
	}
	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		libp2p.Identity(sk),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.ConnectionManager(connmgr),
		libp2p.DefaultTransports,
	)
	if err != nil {
		return nil, err
	}
	return host, nil
}

func makeBootstrapPeer(ctx context.Context) (host.Host, *dht.IpfsDHT, error) {
	host, err := makeHost(ctx, bootstrapListenAddrString)
	if err != nil {
		return nil, nil, err
	}

	idht, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		return host, nil, err
	}

	if err := idht.Bootstrap(ctx); err != nil {
		log.Printf("warn: dht bootstrapping failed: %s\n", err.Error())
	}
	rhost := routedhost.Wrap(host, idht)

	return rhost, idht, nil
}

func connectBootstrapPeer(ctx context.Context, ph host.Host, peers ...peer.AddrInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)
		go func(p peer.AddrInfo) {
			defer wg.Done()
			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Println(ctx, "bootstrapDialFailed", p.ID)
				log.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Printf("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

func makePeer(ctx context.Context, peerSeq int, bootstrapID string) (host.Host, *dht.IpfsDHT, error) {
	port := basePeerPort + peerSeq
	listenAddr := fmt.Sprintf(peerListenAddrStringFormat, port)
	log.Printf("info: new peer with listen addr: %s\n", listenAddr)

	host, err := makeHost(ctx, listenAddr)
	if err != nil {
		return nil, nil, err
	}
	bootstrapAddr := fmt.Sprintf(bootstrapPeerFormat, bootstrapID)
	log.Printf("info: bootstrap peer addr: %s\n", bootstrapAddr)

	bootstrapHosts := convertPeers([]string{bootstrapAddr})

	idht, err := dht.New(ctx, host, dht.BootstrapPeersFunc(func() []peer.AddrInfo {
		return bootstrapHosts
	}))
	if err != nil {
		return host, nil, err
	}

	connectBootstrapPeer(ctx, host, bootstrapHosts...)

	if err := idht.Bootstrap(ctx); err != nil {
		log.Printf("warn: dht bootstrapping failed: %s\n", err.Error())
	}

	rhost := routedhost.Wrap(host, idht)

	return rhost, idht, nil
}

func makeStoragePeer(ctx context.Context, seq int, bsid string) (*storage, host.Host, *dht.IpfsDHT, error) {
	h, d, err := makePeer(ctx, seq, bsid)
	if err != nil {
		return nil, nil, nil, err
	}

	bucket := fmt.Sprintf("peer%d", seq)
	store, storeErr := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir("/tmp"), fsstore.WithBucket(bucket))
	if storeErr != nil {
		return nil, nil, nil, storeErr
	}

	storage := &storage{
		debug:     true,
		chunkSize: defaultChunkSize,
		store:     store,
		host:      h,
		crouter:   d,
	}

	storage.registerPeerProtocol()

	return storage, h, d, nil
}

func TestOneNetworkPeerContextHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bh, bd, berr := makeBootstrapPeer(ctx)
	require.Nil(t, berr)
	require.NotNil(t, bd)
	require.NotNil(t, bh)
	defer bd.Close()
	defer bh.Close()

	p1s, p1h, p1d, p1err := makeStoragePeer(ctx, 1, bh.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	require.NotNil(t, p1h)
	require.NotNil(t, p1d)
	defer p1d.Close()
	defer p1h.Close()

	cid, err := cid.Decode(notExistsCid)
	require.Nil(t, err)

	cancel()
	_, err = p1s.findBlockProvider(ctx, cid)
	require.Equal(t, ErrBlockOperationCancelled, err)

}

func TestOneNetworkPeerNotExistsBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bh, bd, berr := makeBootstrapPeer(ctx)
	require.Nil(t, berr)
	require.NotNil(t, bd)
	require.NotNil(t, bh)
	defer bd.Close()
	defer bh.Close()

	p1s, p1h, p1d, p1err := makeStoragePeer(ctx, 1, bh.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	require.NotNil(t, p1h)
	require.NotNil(t, p1d)
	defer p1d.Close()
	defer p1h.Close()

	cid, err := cid.Decode(notExistsCid)
	require.Nil(t, err)

	require.Equal(t, false, p1s.store.HasObject(ctx, cid))

	_, err = p1s.findBlockProvider(ctx, cid)
	require.Equal(t, ErrBlockProviderNotFound, err)
}

func generateRandomByteReader(size int) io.Reader {
	if size == 0 {
		return bytes.NewReader([]byte{})
	}
	blk := make([]byte, size)
	_, err := rand.Read(blk)
	if err != nil {
		log.Printf("err: error occured while generating random bytes: %s\n", err)
		return bytes.NewReader([]byte{})
	}
	return bytes.NewReader(blk)

}

func TestOneNetworkPeerBlockCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bh, bd, berr := makeBootstrapPeer(ctx)
	require.Nil(t, berr)
	require.NotNil(t, bd)
	require.NotNil(t, bh)
	defer bd.Close()
	defer bh.Close()

	p1s, p1h, p1d, p1err := makeStoragePeer(ctx, 1, bh.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	require.NotNil(t, p1h)
	require.NotNil(t, p1d)
	defer p1d.Close()
	defer p1h.Close()

	fileName := "sample.file"
	totalSize := 512 << 10

	digest, creationErr := p1s.CreateBlock(ctx, fileName, generateRandomByteReader(totalSize))
	require.Nil(t, creationErr)
	require.NotEmpty(t, digest)

	cid, decodeErr := cid.Decode(digest)
	require.Nil(t, decodeErr)

	require.True(t, p1s.store.HasObject(ctx, cid))

	providers, err := p1s.findBlockProvider(ctx, cid)
	require.Nil(t, err)
	require.Equal(t, 1, len(providers))

	provider := providers[0]
	require.Equal(t, p1h.ID(), provider.ID)

	block, err := p1s.GetBlock(ctx, cid)
	require.Nil(t, err)
	require.Nil(t, block.Data)
	require.Equal(t, 1, len(block.Links))
	link := block.Links[0]
	require.Equal(t, fileName, link.Name)
	require.Equal(t, uint64(totalSize), link.Tsize)
}

func TestTwoNetworkPeersBlockFetching(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bh, bd, berr := makeBootstrapPeer(ctx)
	require.Nil(t, berr)
	require.NotNil(t, bd)
	require.NotNil(t, bh)
	defer bd.Close()
	defer bh.Close()

	p1s, p1h, p1d, p1err := makeStoragePeer(ctx, 1, bh.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	require.NotNil(t, p1h)
	require.NotNil(t, p1d)
	defer p1d.Close()
	defer p1h.Close()

	p2s, p2h, p2d, p2err := makeStoragePeer(ctx, 2, bh.ID().String())
	require.Nil(t, p2err)
	require.NotNil(t, p2s)
	require.NotNil(t, p2h)
	require.NotNil(t, p2d)
	defer p2d.Close()
	defer p2h.Close()

	fileName := "sample.file"
	totalSize := 512 << 10

	digest, creationErr := p1s.CreateBlock(ctx, fileName, generateRandomByteReader(totalSize))
	require.Nil(t, creationErr)
	require.NotEmpty(t, digest)

	cid, decodeErr := cid.Decode(digest)
	require.Nil(t, decodeErr)

	require.True(t, p1s.store.HasObject(ctx, cid))

	providers, err := p1s.findBlockProvider(ctx, cid)
	require.Nil(t, err)
	require.Equal(t, 1, len(providers))

	provider := providers[0]
	require.Equal(t, p1h.ID(), provider.ID)

	block, err := p1s.GetBlock(ctx, cid)
	require.Nil(t, err)
	require.Nil(t, block.Data)
	require.Equal(t, 1, len(block.Links))
	link := block.Links[0]
	require.Equal(t, fileName, link.Name)
	require.Equal(t, uint64(totalSize), link.Tsize)

	require.False(t, p2s.store.HasObject(ctx, cid))
	remoteBlock, remoteErr := p2s.GetBlock(ctx, cid)
	require.Nil(t, remoteErr)

	require.Nil(t, remoteBlock.Data)
	require.Equal(t, len(block.Links), len(remoteBlock.Links))

	remoteLink := remoteBlock.Links[0]
	require.Equal(t, link.Hash, remoteLink.Hash)
	require.Equal(t, link.Tsize, remoteLink.Tsize)

}
