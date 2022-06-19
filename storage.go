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

type contextChecker interface {
	checkContext(context.Context) error
}

type BlockStorage interface {
	contextChecker
	CreateBlock(context.Context, string, io.Reader) (string, error)
	GetBlock(context.Context, cid.Cid) (*blockpb.Block, error)
	Stop() error
}

type storage struct {
	debug      bool
	chunkSize  int
	localStore objectstore.ObjectStore
	host       host.Host
	crouter    routing.ContentRouting
}

func NewBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.localStore = cfg.ostore
	ret.debug = cfg.debugMode
	ret.chunkSize = cfg.chunkSize
	ret.host = cfg.peerHost
	ret.crouter = cfg.peerContentRouter

	ret.registerPeerProtocol()
	ret.registerGrpc(cfg.grpcServer)

	return ret, nil
}

func (s *storage) checkContext(ctx context.Context) error {
	if ctx.Err() == nil {
		return nil
	}
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return ErrBlockOperationTimedOut
	default:
		return ErrBlockOperationCancelled
	}
}

func (s *storage) Stop() error {
	log.Println("info: blockstorage service stopped")
	return nil
}
