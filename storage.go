package blockstorage

import (
	"context"
	"io"
	"log"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/go-objectstore-lib"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
)

type BlockStorage interface {
	CreateBlock(context.Context, string, io.Reader) (string, error)
	GetBlock(context.Context, cid.Cid) (*blockpb.Block, error)
	Stop() error
}

type storage struct {
	debug     bool
	chunkSize int
	store     objectstore.ObjectStore
	host      host.Host
	crouter   routing.ContentRouting
}

func NewBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.store = cfg.ostore
	ret.debug = cfg.debugMode
	ret.chunkSize = cfg.chunkSize
	ret.host = cfg.peerHost
	ret.crouter = cfg.peerContentRouter

	ret.registerPeerProtocol()
	ret.registerGrpc(cfg.grpcServer)

	return ret, nil
}

func (s *storage) Stop() error {
	log.Println("info: blockstorage service stopped")
	return nil
}
