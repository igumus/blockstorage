package blockstorage

import (
	"context"
	"log"

	"github.com/igumus/go-objectstore-lib"
)

type BlockStorage interface {
	Start() error
	Stop() error
}

type storage struct {
	debug bool
	store objectstore.ObjectStore
}

func NewBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.store = cfg.ostore
	ret.debug = cfg.debugMode

	return ret, nil
}

func (s *storage) Start() error {
	log.Println("info: blockstorage service started")
	return nil
}

func (s *storage) Stop() error {
	log.Println("info: blockstorage service stopped")
	return nil
}
