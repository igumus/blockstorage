package blockstorage

import (
	"context"
	"io"
	"log"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/blockstorage/peer"
	"github.com/igumus/go-objectstore-lib"
	"github.com/ipfs/go-cid"
)

// Defines/Represents block storage's public functionality
type BlockStorage interface {
	CreateBlock(context.Context, string, io.Reader) (string, error)
	GetBlock(context.Context, cid.Cid) (*blockpb.Block, error)
	Stop() error
}

// Captures/Represents block storage's internal structure
type storage struct {
	debug      bool
	chunkSize  int
	localStore objectstore.ObjectStore
	peer       peer.BlockStoragePeer
}

// NewFakeBlockStorage - creates a new `BlockStorage` instance for mocking.
// - Registering peer Read Protocol disabled.
// DO NOT USE AS REAL INSTANCE.
func NewFakeBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.debug = cfg.debugMode

	ret.localStore = cfg.lstore
	ret.peer = cfg.peer

	ret.chunkSize = cfg.chunkSize

	return ret, nil
}

// NewBlockStorage - creates a new `BlockStorage` instace. If given options are valid returns the instance.
// Otherwise return validation error
func NewBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.debug = cfg.debugMode

	ret.localStore = cfg.lstore
	ret.peer = cfg.peer
	ret.peer.RegisterReadProtocol(ctx, ret.localStore)

	ret.chunkSize = cfg.chunkSize

	return ret, nil
}

func (s *storage) Stop() error {
	log.Println("info: blockstorage service stopped")
	return nil
}
