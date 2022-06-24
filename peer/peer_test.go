package peer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/go-objectstore-lib/mock"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"

	libpeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

func convertPeers(peers []string) []libpeer.AddrInfo {
	pinfos := make([]libpeer.AddrInfo, len(peers))
	for i, addr := range peers {
		maddr := ma.StringCast(addr)
		p, err := libpeer.AddrInfoFromP2pAddr(maddr)
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

func connectBootstrapPeer(ctx context.Context, ph host.Host, peers ...libpeer.AddrInfo) error {
	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)
		go func(p libpeer.AddrInfo) {
			defer wg.Done()
			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Println(ctx, "bootstrapDialFailed", p.ID)
				log.Printf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
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

	idht, err := dht.New(ctx, host, dht.BootstrapPeersFunc(func() []libpeer.AddrInfo {
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

type peerSuite struct {
	suite.Suite
	*require.Assertions

	ctrl          *gomock.Controller
	bootstrapHost host.Host
	digestPrefix  cid.Prefix
}

func TestPeerSuite(t *testing.T) {
	suite.Run(t, new(peerSuite))
}

func (s *peerSuite) SetupTest() {
	s.Assertions = require.New(s.T())
	s.ctrl = gomock.NewController(s.T())
	h, err := makeBootstrapPeer(context.Background())
	require.NoError(s.T(), err)
	s.bootstrapHost = h
	s.digestPrefix = cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   mh.SHA2_256,
		MhLength: -1,
	}
}

func (s *peerSuite) TearDownTest() {
	s.ctrl.Finish()
	s.bootstrapHost.Close()
}

func (s *peerSuite) TestAskingNotExistedBlock() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blockID, err := cid.Decode(notExistsCid)
	require.NoError(s.T(), err)

	h1, dht1, err := makePeer(ctx, 1, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht1)
	require.NotNil(s.T(), h1)
	defer dht1.Close()
	defer h1.Close()

	temporaryStore1 := mock.NewMockObjectStore(s.ctrl)
	_, err = newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(1), WithContentRouter(dht1), WithHost(h1), WithTempStore(temporaryStore1))
	require.NoError(s.T(), err)

	h2, dht2, err := makePeer(ctx, 2, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht2)
	require.NotNil(s.T(), h2)
	defer dht2.Close()
	defer h2.Close()

	temporaryStore2 := mock.NewMockObjectStore(s.ctrl)
	temporaryStore2.EXPECT().HasObject(gomock.Any(), blockID).Times(2).Return(false)
	peer2, err := newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(3), WithContentRouter(dht2), WithHost(h2), WithTempStore(temporaryStore2))
	require.NoError(s.T(), err)

	_, err = peer2.GetRemoteBlock(ctx, blockID)
	require.NotNil(s.T(), err)
	require.Error(s.T(), ErrBlockProviderNotFound, err)
	require.False(s.T(), peer2.store.HasObject(ctx, blockID))
}

func (s *peerSuite) TestSharingDataBlock() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	block := &blockpb.Block{Data: []byte("selam")}
	bin, err := blockpb.Encode(block)
	require.NoError(s.T(), err)

	blockID, err := s.digestPrefix.Sum(bin)
	require.NoError(s.T(), err)

	h1, dht1, err := makePeer(ctx, 1, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht1)
	require.NotNil(s.T(), h1)
	defer dht1.Close()
	defer h1.Close()

	permanentStore1 := mock.NewMockObjectStore(s.ctrl)
	temporaryStore1 := mock.NewMockObjectStore(s.ctrl)
	peer1, err := newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(1), WithContentRouter(dht1), WithHost(h1), WithTempStore(temporaryStore1))
	require.NoError(s.T(), err)
	peer1.RegisterReadProtocol(ctx, permanentStore1)
	announced := peer1.AnnounceBlock(ctx, blockID)
	require.True(s.T(), announced)

	permanentStore1.EXPECT().ReadObject(gomock.Any(), blockID).Times(1).Return(bin, nil)

	h2, dht2, err := makePeer(ctx, 2, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht2)
	require.NotNil(s.T(), h2)
	defer dht2.Close()
	defer h2.Close()
	fsmap := make(map[cid.Cid]bool)

	temporaryStore2 := mock.NewMockObjectStore(s.ctrl)
	temporaryStore2.EXPECT().HasObject(gomock.Any(), blockID).Times(2).DoAndReturn(func(_ context.Context, id cid.Cid) bool {
		return fsmap[id]
	})
	temporaryStore2.EXPECT().CreateObject(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ io.Reader) (cid.Cid, error) {
		fsmap[blockID] = true
		return blockID, nil
	})
	peer2, err := newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(1), WithContentRouter(dht2), WithHost(h2), WithTempStore(temporaryStore2))
	require.NoError(s.T(), err)

	_, err = peer2.GetRemoteBlock(ctx, blockID)
	require.Nil(s.T(), err)
	require.True(s.T(), peer2.store.HasObject(ctx, blockID))
}

func (s *peerSuite) TestFetchingSubNodes() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	child1 := &blockpb.Block{Data: []byte("selam1")}
	bin1, err := blockpb.Encode(child1)
	require.NoError(s.T(), err)
	child1_ID, err := s.digestPrefix.Sum(bin1)
	require.NoError(s.T(), err)

	child2 := &blockpb.Block{Data: []byte("selam2")}
	bin2, err := blockpb.Encode(child2)
	require.NoError(s.T(), err)
	child2_ID, err := s.digestPrefix.Sum(bin2)
	require.NoError(s.T(), err)

	links := append([]*blockpb.Link{}, &blockpb.Link{Hash: child1_ID.String()}, &blockpb.Link{Hash: child2_ID.String()})

	block := &blockpb.Block{Name: "selams.txt", Links: links}
	bin, err := blockpb.Encode(block)
	require.NoError(s.T(), err)
	blockID, err := s.digestPrefix.Sum(bin)
	require.NoError(s.T(), err)

	h1, dht1, err := makePeer(ctx, 1, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht1)
	require.NotNil(s.T(), h1)
	defer dht1.Close()
	defer h1.Close()

	permanentStore1 := mock.NewMockObjectStore(s.ctrl)
	temporaryStore1 := mock.NewMockObjectStore(s.ctrl)
	peer1, err := newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(1), WithContentRouter(dht1), WithHost(h1), WithTempStore(temporaryStore1))
	require.NoError(s.T(), err)
	peer1.RegisterReadProtocol(ctx, permanentStore1)
	announced := peer1.AnnounceBlock(ctx, blockID)
	require.True(s.T(), announced)

	permanentStore1.EXPECT().ReadObject(gomock.Any(), child1_ID).Times(1).Return(bin1, nil)
	permanentStore1.EXPECT().ReadObject(gomock.Any(), child2_ID).Times(1).Return(bin2, nil)
	permanentStore1.EXPECT().ReadObject(gomock.Any(), blockID).Times(1).Return(bin, nil)

	h2, dht2, err := makePeer(ctx, 2, s.bootstrapHost.ID().String())
	require.NoError(s.T(), err)
	require.NotNil(s.T(), dht2)
	require.NotNil(s.T(), h2)
	defer dht2.Close()
	defer h2.Close()
	fsmap := make(map[cid.Cid]bool)

	temporaryStore2 := mock.NewMockObjectStore(s.ctrl)
	temporaryStore2.EXPECT().HasObject(gomock.Any(), gomock.Any()).Times(4).DoAndReturn(func(_ context.Context, id cid.Cid) bool {
		return fsmap[id]
	})
	temporaryStore2.EXPECT().CreateObject(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(func(_ context.Context, r io.Reader) (cid.Cid, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(s.T(), err)
		id, err := s.digestPrefix.Sum(data)
		require.NoError(s.T(), err)

		fsmap[id] = true
		return id, nil
	})
	peer2, err := newBlockStoragePeer(ctx, EnableDebugMode(), WithMaxProviderCount(1), WithContentRouter(dht2), WithHost(h2), WithTempStore(temporaryStore2))
	require.NoError(s.T(), err)

	_, err = peer2.GetRemoteBlock(ctx, blockID)
	require.Nil(s.T(), err)
	require.True(s.T(), peer2.store.HasObject(ctx, blockID))
	require.True(s.T(), peer2.store.HasObject(ctx, child1_ID))
	require.True(s.T(), peer2.store.HasObject(ctx, child2_ID))
}
