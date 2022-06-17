package blockstorage

import (
	"github.com/igumus/go-objectstore-lib"
)

// A BlockStorageOption sets options.
type BlockStorageOption func(*blockstorageConfig)

// Captures/Represents BlockStorage's configuration information.
type blockstorageConfig struct {
	ostore    objectstore.ObjectStore
	debugMode bool
}

// validate - validates given `blockstorageConfig` instance
func validate(s *blockstorageConfig) error {
	return nil
}

// defaultBlockstorageConfig - returns instance of `blockstorageConfig` with initial values.
func defaultBlockstorageConfig() *blockstorageConfig {
	return &blockstorageConfig{
		ostore:    nil,
		debugMode: false,
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

// WithObjectStore returns a BlockStorageOption that specifies objectstore as persistence storage
func WithObjectStore(s objectstore.ObjectStore) BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.ostore = s
	}
}

// EnableDebugMode returns a BlockStorageOption that enabled debug mode for BlockStorage service
func EnableDebugMode() BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.debugMode = true
	}
}
