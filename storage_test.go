package blockstorage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const dataDir = "/tmp"

var dataDirOption = fsstore.WithDataDir(dataDir)

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

func makeBootstrapPeer(ctx context.Context) (host.Host, error) {
	host, err := makeHost(ctx, bootstrapListenAddrString)
	if err != nil {
		return nil, err
	}

	idht, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		return host, err
	}

	if err := idht.Bootstrap(ctx); err != nil {
		log.Printf("warn: dht bootstrapping failed: %s\n", err.Error())
	}
	rhost := routedhost.Wrap(host, idht)

	return rhost, nil
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

func makeStoragePeer(ctx context.Context, seq int, bsid string, otherOpts ...BlockStorageOption) (*storage, func(), error) {
	h, d, err := makePeer(ctx, seq, bsid)
	if err != nil {
		return nil, nil, err
	}

	bucket := fmt.Sprintf("peer%d", seq)
	bucketTmp := fmt.Sprintf("peer%d-temp", seq)
	store, storeErr := fsstore.NewFileSystemObjectStore(dataDirOption, fsstore.WithBucket(bucket))
	if storeErr != nil {
		return nil, nil, storeErr
	}
	tStore, tStoreErr := fsstore.NewFileSystemObjectStore(dataDirOption, fsstore.WithBucket(bucketTmp))
	if tStoreErr != nil {
		return nil, nil, tStoreErr
	}

	opts := []BlockStorageOption{
		EnableDebugMode(),
		WithLocalStore(store),
		WithTempStore(tStore),
		WithPeer(h, d),
	}
	if len(otherOpts) > 0 {
		opts = append(opts, otherOpts...)
	}

	s, serr := NewBlockStorage(ctx, opts...)
	if serr != nil {
		return nil, nil, serr
	}
	x := s.(*storage)
	return x, func() {
		d.Close()
		h.Close()
		os.RemoveAll(dataDir + bucket)
		os.RemoveAll(dataDir + bucketTmp)
	}, nil
}

func makeGrpcServer() (*grpc.Server, *bufconn.Listener, func(), func()) {
	s := grpc.NewServer()
	lis := bufconn.Listen(bufSize)
	return s, lis, func() {
			if err := s.Serve(lis); err != nil {
				log.Fatalf("Server exited with error: %v", err)
			}
		}, func() {
			s.GracefulStop()
		}
}

const notExistsCid = "bafkreicbhkvymvquwrtgsxbed6imq5ec6526it55c3kp5lxpcjujyg7a4m"
const bootstrapListenAddrString = "/ip4/127.0.0.1/tcp/3001"
const bootstrapPeerFormat = bootstrapListenAddrString + "/p2p/%s"

const basePeerPort = 5000
const peerListenAddrStringFormat = "/ip4/127.0.0.1/tcp/%d"
const bufSize = 1024 * 1024
const chunkSize = 512 << 10

var bootstrapHost host.Host

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	ctx := context.Background()
	var err error
	bootstrapHost, err = makeBootstrapPeer(ctx)
	if err != nil {
		log.Fatalf("testingErr: creating bootstrap peer failed: %s\n", err.Error())
	}

	fmt.Printf("\033[1;36m%s\033[0m", "> Setup completed\n")
}

func teardown() {
	if bootstrapHost != nil {
		bootstrapHost.Close()
	}
	fmt.Printf("\033[1;36m%s\033[0m", "> Teardown completed\n")
}
