package blockstorage

import (
	"errors"

	"github.com/igumus/blockstorage/peer"
	"github.com/igumus/go-objectstore-lib"
)

// ErrLocalObjectStoreNotDefined is return when local objectstore not specified while constructing `BlockStorage` service
var ErrLocalObjectStoreNotDefined = errors.New("[blockstorage] block storage configuration failed: permanent store instance not specified")

// ErrPeerNotSpecified is return when peer not specified while constructing `BlockStorage` service
var ErrPeerNotSpecified = errors.New("[blockstorage] block storage configuration failed: peer instance not specified")

// defaultChunkSize handles default size in KB
const defaultChunkSize = 512 << 10

// A BlockStorageOption sets options.
type BlockStorageOption func(*blockstorageConfig)

// Captures/Represents BlockStorage's configuration information.
type blockstorageConfig struct {
	lstore    objectstore.ObjectStore
	debugMode bool
	chunkSize int
	peer      peer.BlockStoragePeer
}

// validate - validates given `blockstorageConfig` instance
func validate(s *blockstorageConfig) error {
	if s.lstore == nil {
		return ErrLocalObjectStoreNotDefined
	}
	if s.peer == nil {
		return ErrPeerNotSpecified
	}
	return nil
}

// defaultBlockstorageConfig - returns instance of `blockstorageConfig` with initial values.
func defaultBlockstorageConfig() *blockstorageConfig {
	return &blockstorageConfig{
		lstore:    nil,
		peer:      nil,
		debugMode: false,
		chunkSize: defaultChunkSize,
	}
}

// createConfig - creates new `blockstorageConfig` with given options.
// Creates default configuration and applys options to configuration.
// Returns configuration instance and validation result.
func createConfig(opts ...BlockStorageOption) (*blockstorageConfig, error) {
	cfg := defaultBlockstorageConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg, validate(cfg)
}

// WithLocalStore returns a BlockStorageOption that specifies object store as permanent store.
func WithLocalStore(s objectstore.ObjectStore) BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.lstore = s
	}
}

// WithPeer returns a BlockStorageOption that specifies peer host and peer content router to satify p2p capabilities
func WithPeer(p peer.BlockStoragePeer) BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.peer = p
	}
}

// EnableDebugMode returns a BlockStorageOption that enabled debug mode for BlockStorage service
func EnableDebugMode() BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.debugMode = true
	}
}
