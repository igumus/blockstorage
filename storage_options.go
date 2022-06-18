package blockstorage

import (
	"github.com/igumus/go-objectstore-lib"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	"google.golang.org/grpc"
)

// defaultChunkSize handles default size in KB
const defaultChunkSize = 512 << 10

// A BlockStorageOption sets options.
type BlockStorageOption func(*blockstorageConfig)

// Captures/Represents BlockStorage's configuration information.
type blockstorageConfig struct {
	ostore            objectstore.ObjectStore
	grpcServer        *grpc.Server
	peerHost          host.Host
	peerContentRouter routing.ContentRouting
	debugMode         bool
	chunkSize         int
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

// WithPeer returns a BlockStorageOption that specifies peer host and peer content router to satify p2p capabilities
func WithPeer(h host.Host, r routing.ContentRouting) BlockStorageOption {
	return func(bc *blockstorageConfig) {
		bc.peerHost = h
		bc.peerContentRouter = r
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
