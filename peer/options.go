package peer

import (
	"errors"

	"github.com/igumus/go-objectstore-lib"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
)

var ErrPeerHostNotSpecified = errors.New("[blockstorage] peer configuration failed: host not specified")

var ErrPeerContentRouterNotSpecified = errors.New("[blockstorage] peer configuration failed: content router not specified")

var ErrPeerMaxProviderCountInvalid = errors.New("[blockstorage] peer configuration failed: max provider count should be at least 1")

var ErrPeerTemporaryStoreNotSpecified = errors.New("[blockstorage] peer configuration failed: temporary store not specified")

// defaultMaxProviderCount holds how many provider to ask max while finding block provider
const defaultMaxProviderCount = 3

// A PeerOption sets options.
type PeerOption func(*peerConfig)

// Captures/Represents BlockStoragePeer's configuration information.
type peerConfig struct {
	debugMode        bool
	store            objectstore.ObjectStore
	host             host.Host
	contentRouter    routing.ContentRouting
	maxProviderCount int
}

// validate - validates given `peerConfig` instance
func validate(s *peerConfig) error {
	if s.host == nil {
		return ErrPeerHostNotSpecified
	}
	if s.contentRouter == nil {
		return ErrPeerContentRouterNotSpecified
	}
	if s.maxProviderCount < 1 {
		return ErrPeerMaxProviderCountInvalid
	}
	if s.store == nil {
		return ErrPeerTemporaryStoreNotSpecified
	}
	return nil
}

// defaultPeerConfig - returns instance of `peerConfig` with initial values.
func defaultPeerConfig() *peerConfig {
	return &peerConfig{
		store:            nil,
		host:             nil,
		contentRouter:    nil,
		maxProviderCount: defaultMaxProviderCount,
		debugMode:        false,
	}
}

// createConfig - creates new `peerConfig` with given options.
// Creates default configuration and applys options to configuration.
// Returns configuration instance and validation result.
func createConfig(opts ...PeerOption) (*peerConfig, error) {
	cfg := defaultPeerConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg, validate(cfg)
}

// WithMaxProviderCount returns a PeerOption that specifies how many remote provider to receive block result.
// If not specified default value is 3
func WithMaxProviderCount(p int) PeerOption {
	return func(pc *peerConfig) {
		pc.maxProviderCount = p
	}
}

// WithTempStore returns a PeerOption that specifies object store as temporary store.
// If not specified any, uses noop store (which has no persistence).
func WithTempStore(s objectstore.ObjectStore) PeerOption {
	return func(bc *peerConfig) {
		bc.store = s
	}
}

// WithHost returns a PeerOption that specifies libp2p host.
func WithHost(h host.Host) PeerOption {
	return func(pc *peerConfig) {
		pc.host = h
	}
}

// WithContentRouter returns a PeerOption that specifies libp2p contentRouting.
func WithContentRouter(cr routing.ContentRouting) PeerOption {
	return func(pc *peerConfig) {
		pc.contentRouter = cr
	}
}

// EnableDebugMode returns a PeerOption that enables debug mode
func EnableDebugMode() PeerOption {
	return func(pc *peerConfig) {
		pc.debugMode = true
	}
}
