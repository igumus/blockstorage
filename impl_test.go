package blockstorage

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	mockpeer "github.com/igumus/blockstorage/peer/mock"
	"github.com/igumus/go-objectstore-lib"
	"github.com/igumus/go-objectstore-lib/mock"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func (s *blockStorageSuite) TestBlockCreation() {
	ctx := context.Background()
	testCases := []struct {
		name          string
		expectedName  string
		data          io.Reader
		shouldFail    bool
		blockLinkSize int
		err           error
	}{
		{
			name:          "valid_input",
			expectedName:  "valid_input",
			data:          generateRandomByteReader(s.T(), 3),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "equal_to_chunk_size",
			expectedName:  "equal_to_chunk_size",
			data:          generateRandomByteReader(s.T(), 512<<10),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "double_chunk_size",
			expectedName:  "double_chunk_size",
			data:          generateRandomByteReader(s.T(), 1024<<10),
			blockLinkSize: 2,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          " spaced_name ",
			expectedName:  "spaced_name",
			data:          generateRandomByteReader(s.T(), 3),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "empty_data",
			data:          generateRandomByteReader(s.T(), 0),
			blockLinkSize: 0,
			shouldFail:    true,
			err:           ErrBlockDataEmpty,
		},
		{
			name:          "",
			data:          generateRandomByteReader(s.T(), 3),
			blockLinkSize: 1,
			shouldFail:    true,
			err:           ErrBlockNameEmpty,
		},
		{
			name:          " ",
			data:          generateRandomByteReader(s.T(), 3),
			blockLinkSize: 0,
			shouldFail:    true,
			err:           ErrBlockNameEmpty,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		s.T().Run(tc.name, func(t *testing.T) {
			store := mock.NewMockObjectStore(s.ctrl)
			peer := mockpeer.NewMockBlockStoragePeer(s.ctrl)
			peer.EXPECT().AnnounceBlock(gomock.Any(), gomock.Any()).AnyTimes().Return(true)

			storage, err := NewFakeBlockStorage(ctx, WithLocalStore(store), WithPeer(peer))
			require.NoError(s.T(), err)

			lookup := make(map[cid.Cid][]byte)

			if !tc.shouldFail {
				store.EXPECT().HasObject(gomock.Any(), gomock.Any()).AnyTimes().Return(true)
				store.EXPECT().CreateObject(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, r io.Reader) (cid.Cid, error) {
					data, err := ioutil.ReadAll(r)
					require.NoError(s.T(), err)

					id, err := objectstore.DigestPrefix.Sum(data)
					require.NoError(s.T(), err)
					lookup[id] = data
					return id, nil
				})
				store.EXPECT().ReadObject(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, id cid.Cid) ([]byte, error) {
					return lookup[id], nil
				})
			}

			digest, createErr := storage.CreateBlock(ctx, tc.name, tc.data)

			if tc.shouldFail {
				require.NotNil(s.T(), createErr)
				require.Equal(s.T(), createErr, tc.err)
			} else {
				require.Nil(t, createErr)
				rootCid, err := cid.Decode(digest)
				require.NoError(s.T(), err)

				rootBlock, readErr := storage.GetBlock(ctx, rootCid)
				require.Nil(t, readErr)
				require.Nil(t, rootBlock.Data)
				require.Equal(t, tc.blockLinkSize, len(rootBlock.Links))
				require.Equal(t, tc.expectedName, rootBlock.Name)
			}

		})
	}
}
