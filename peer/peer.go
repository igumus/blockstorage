//go:generate mockgen -source peer.go -destination mock/peer_mock.go -package mock
package peer

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"sync"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/blockstorage/util"
	"github.com/igumus/go-objectstore-lib"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	libpeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
)

// ErrBlockProviderNotFound is return, when there is no owner of specified block.
var ErrBlockProviderNotFound = errors.New("blockstorage: not found any provider for block")

type BlockStoragePeer interface {
	RegisterReadProtocol(context.Context, objectstore.ObjectStore)
	AnnounceBlock(context.Context, cid.Cid) bool
	GetRemoteBlock(context.Context, cid.Cid) ([]byte, error)
}

type peer struct {
	debug            bool
	host             host.Host
	contentRouter    routing.ContentRouting
	store            objectstore.ObjectStore
	maxProviderCount int
}

func newBlockStoragePeer(ctx context.Context, opts ...PeerOption) (*peer, error) {
	cfg, err := createConfig(opts...)
	if err != nil {
		return nil, err
	}
	ret := &peer{
		debug:            cfg.debugMode,
		host:             cfg.host,
		contentRouter:    cfg.contentRouter,
		store:            cfg.store,
		maxProviderCount: cfg.maxProviderCount,
	}
	return ret, nil
}

func NewBlockStoragePeer(ctx context.Context, opts ...PeerOption) (BlockStoragePeer, error) {
	return newBlockStoragePeer(ctx, opts...)
}

func (p *peer) RegisterReadProtocol(ctx context.Context, store objectstore.ObjectStore) {
	p.host.SetStreamHandler(BlockReadProtocolID, generateReadProtocol(store))
}

// AnnounceBlock - announces ownership of given cid (aka content identifier) to the p2p network.
// Returns `true` in successful announcement, otherwise `false`
func (p *peer) AnnounceBlock(ctx context.Context, blockID cid.Cid) bool {
	if err := p.contentRouter.Provide(ctx, blockID, true); err != nil {
		log.Printf("warn: announcing block failed: %s, %s\n", blockID, err.Error())
		return false
	}
	log.Printf("info: announcing block succeed: %s\n", blockID)
	return true
}

// findBlockProvider - searches ownership of given cid (aka content identifier) on the p2p network.
// If found any provider, returns address information of that peer(s).
// Otherwise returns `ErrBlockProviderNotFound` error.
func (p *peer) findBlockProvider(ctx context.Context, blockID cid.Cid) ([]libpeer.AddrInfo, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	chProviders := p.contentRouter.FindProvidersAsync(ctx, blockID, p.maxProviderCount)
	providers := make([]libpeer.AddrInfo, 0, p.maxProviderCount)
	for provider := range chProviders {
		providers = append(providers, provider)
	}
	ctxErr = util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	if len(providers) > 0 {
		return providers, nil
	}

	return nil, ErrBlockProviderNotFound
}

// fetchRemoteBlock - fetches given cid (aka content identifier) from remote peer
// While fetching creates 1:1 stream with the remote peer.
// On succesful communication returns, byte content of desired block, otherwise returns cause error
func (p *peer) fetchRemoteBlock(ctx context.Context, blockID cid.Cid, peerAddr libpeer.AddrInfo) ([]byte, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	log.Printf("info: fetching object %s from %s\n", blockID, peerAddr.ID)
	stream, err := p.host.NewStream(ctx, peerAddr.ID, BlockReadProtocolID)
	if err != nil {
		log.Printf("err: creating stream failed: %s, %s\n", peerAddr, err.Error())
		return nil, err
	}
	defer stream.Close()

	bin, err := blockID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(bin)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(stream)

	newCid, createErr := p.store.CreateObject(ctx, bytes.NewReader(data))
	if createErr != nil {
		log.Printf("err: storing remote block to temp store failed: %s, %s\n", blockID, createErr.Error())
	} else {
		log.Printf("info: requested block:%s, received block: %s\n", blockID, newCid)
	}

	return data, err
}

// GetRemoteBlock - gets remote block with given cid (aka content identifier) from p2p network.
//
// Flow:
// 1. Finds provider for given block cid
// 2. Fetches block from found provider (currently first provider) via `/blockstorage/block/read/1.0.0` peer protocol
// 3. Persists fetched block to temporary object store.
// 4. Returns encoded/marshalled block
//
// Error:
// When any of the flow operations fail, returns `nil` with error cause
func (p *peer) GetRemoteBlock(ctx context.Context, blockID cid.Cid) ([]byte, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}

	if p.store.HasObject(ctx, blockID) {
		if p.debug {
			log.Printf("debug: block already in temporary store: %s\n", blockID)
		}
		return p.store.ReadObject(ctx, blockID)
	}

	providers, err := p.findBlockProvider(ctx, blockID)
	if err != nil {
		return nil, err
	}
	provider := providers[0]

	data, err := p.fetchRemoteBlock(ctx, blockID, provider)
	if err != nil {
		return nil, err
	}

	block, blockErr := blockpb.Decode(data)
	if blockErr != nil {
		log.Printf("err: decoding block failed: %s, %s\n", blockID, blockErr.Error())
		return nil, blockErr
	}

	if len(block.Links) > 0 {
		wg := sync.WaitGroup{}
		wg.Add(len(block.Links))
		for _, link := range block.Links {
			go func(l *blockpb.Link) {
				defer wg.Done()
				childCid, err := cid.Decode(l.Hash)
				if err != nil {
					log.Printf("err: decoding child cid failed: %s, %s\n", l.Hash, err.Error())
				}
				if _, err := p.fetchRemoteBlock(ctx, childCid, provider); err != nil {
					log.Printf("err: fetching remote object failed: %s, %s\n", childCid, err.Error())
				}
			}(link)
		}
		wg.Wait()
	}

	return data, nil
}
