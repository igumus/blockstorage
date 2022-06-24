package blockstorage

import (
	"testing"

	mockpeer "github.com/igumus/blockstorage/peer/mock"
	"github.com/igumus/go-objectstore-lib/mock"
	"github.com/stretchr/testify/require"
)

func (s *blockStorageSuite) TestBlockStorageConfigCreation() {
	store := mock.NewMockObjectStore(s.ctrl)
	peer := mockpeer.NewMockBlockStoragePeer(s.ctrl)
	testCases := []struct {
		name       string
		shouldFail bool
		options    []BlockStorageOption
		err        error
	}{
		{
			name:       "empty_options",
			options:    make([]BlockStorageOption, 0),
			shouldFail: true,
			err:        ErrLocalObjectStoreNotDefined,
		},
		{
			name:       "without_peer",
			options:    append([]BlockStorageOption{}, WithLocalStore(store)),
			shouldFail: true,
			err:        ErrPeerNotSpecified,
		},
		{
			name:       "valid_options",
			options:    append([]BlockStorageOption{}, WithLocalStore(store), WithPeer(peer)),
			shouldFail: false,
			err:        nil,
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
