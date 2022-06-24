package peer

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/stretchr/testify/require"
)

func makeConfigTestPeer(t *testing.T, includeDHT bool) []PeerOption {
	options := make([]PeerOption, 0, 3)
	options = append(options, EnableDebugMode())
	host, err := libp2p.New()
	require.NoError(t, err)
	defer host.Close()
	options = append(options, WithHost(host))

	if includeDHT {
		idht, err := dht.New(context.Background(), host)
		require.NoError(t, err)
		defer idht.Close()
		options = append(options, WithContentRouter(idht))
	}

	return options
}

func (s *peerSuite) TestPeerConfigCreation() {
	testCases := []struct {
		name       string
		shouldFail bool
		options    []PeerOption
		err        error
	}{
		{
			name:       "empty_options",
			options:    make([]PeerOption, 0),
			shouldFail: true,
			err:        ErrPeerHostNotSpecified,
		},
		{
			name:       "with_NoContentRouter_option",
			options:    makeConfigTestPeer(s.T(), false),
			shouldFail: true,
			err:        ErrPeerContentRouterNotSpecified,
		},
		{
			name:       "with_no_store",
			options:    makeConfigTestPeer(s.T(), true),
			shouldFail: true,
			err:        ErrPeerTemporaryStoreNotSpecified,
		},
		{
			name:       "with_zero_providerCount",
			options:    append(makeConfigTestPeer(s.T(), true), WithMaxProviderCount(0)),
			shouldFail: true,
			err:        ErrPeerMaxProviderCountInvalid,
		},
		{
			name:       "with_negative_providerCount",
			options:    append(makeConfigTestPeer(s.T(), true), WithMaxProviderCount(-1)),
			shouldFail: true,
			err:        ErrPeerMaxProviderCountInvalid,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		s.T().Run(tc.name, func(t *testing.T) {
			cfg, err := createConfig(tc.options...)
			if !tc.shouldFail {
				require.NotNil(s.T(), cfg)
			}
			require.Equal(s.T(), tc.err, err)
		})
	}

}
