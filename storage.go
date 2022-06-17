package blockstorage

import (
	"context"

	"github.com/igumus/go-objectstore-lib"
)

type BlockStorage interface {
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
