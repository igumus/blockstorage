package blockstorage

import (
	"errors"

	"github.com/igumus/go-objectstore-lib"
	"google.golang.org/grpc"
)

var ErrObjectstoreNotDefined = errors.New("blockstorage: objectstore instance not specified")

// defaultChunkSize handles default size in KB
const defaultChunkSize = 512 << 10

// A BlockStorageOption sets options.
type BlockStorageOption func(*blockstorageConfig)

// Captures/Represents BlockStorage's configuration information.
type blockstorageConfig struct {
	ostore     objectstore.ObjectStore
	grpcServer *grpc.Server
	debugMode  bool
	chunkSize  int
}

// validate - validates given `blockstorageConfig` instance
func validate(s *blockstorageConfig) error {
	if s.ostore == nil {
		return ErrObjectstoreNotDefined
	}
	return nil
}

// defaultBlockstorageConfig - returns instance of `blockstorageConfig` with initial values.
func defaultBlockstorageConfig() *blockstorageConfig {
	return &blockstorageConfig{
		ostore:    nil,
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

// EnableGrpcEndpoint returns a BlockStorageOption that enables grpc endpoint of blockstorage
func EnableGrpcEndpoint(s *grpc.Server) BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.grpcServer = s
	}
}
