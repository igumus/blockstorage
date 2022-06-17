package blockstorage

import (
	"context"

	"github.com/igumus/go-objectstore-lib"
)

type BlockStorage interface {
}

type storage struct {
	store objectstore.ObjectStore
}

func NewBlockStorage(ctx context.Context, opts ...BlockStorageOption) (BlockStorage, error) {
	ret := &storage{}

	cfg, cfgErr := createConfig(opts...)
	if cfgErr != nil {
		return ret, cfgErr
	}

	ret.store = cfg.ostore

	return ret, nil
}
