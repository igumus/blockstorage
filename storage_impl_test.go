package blockstorage

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestBlockStorageCreationWithNoStores(t *testing.T) {
	_, storageErr := NewBlockStorage(context.Background())
	require.NotNil(t, storageErr)
	require.Equal(t, storageErr, ErrLocalObjectStoreNotDefined)
}

func TestBlockStorageCreationWithOutTempStore(t *testing.T) {
	folderName := "peer1234-local"
	localStore, err := fsstore.NewFileSystemObjectStore(dataDirOption, fsstore.WithBucket(folderName))
	require.NoError(t, err)
	_, storageErr := NewBlockStorage(context.Background(), WithLocalStore(localStore))
	require.Equal(t, storageErr, ErrTempObjectStoreNotDefined)
	os.RemoveAll(folderName)
}

func TestBlockStorageCreationWithOutPeer(t *testing.T) {
	localFolderName := "peer1234-local"
	tempFolderName := "peer1234-temp"
	localStore, err := fsstore.NewFileSystemObjectStore(dataDirOption, fsstore.WithBucket(localFolderName))
	require.NoError(t, err)
	tempStore, err := fsstore.NewFileSystemObjectStore(dataDirOption, fsstore.WithBucket(tempFolderName))
	require.NoError(t, err)
	_, storageErr := NewBlockStorage(context.Background(), WithLocalStore(localStore), WithTempStore(tempStore))
	require.NoError(t, storageErr)
	os.RemoveAll(localFolderName)
	os.RemoveAll(tempFolderName)
}

func TestBlockCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ps, pshutdown, perr := makeStoragePeer(ctx, 1, bootstrapHost.ID().String())
	require.NoError(t, perr)
	defer pshutdown()

	testCases := []struct {
		name          string
		data          io.Reader
		shouldFail    bool
		blockLinkSize int
		err           error
	}{
		{
			name:          "valid_input",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "equal_to_chunk_size",
			data:          generateRandomByteReader(512 << 10),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "double_chunk_size",
			data:          generateRandomByteReader(1024 << 10),
			blockLinkSize: 2,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          " spaced_name ",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    false,
			err:           nil,
		},
		{
			name:          "empty_data",
			data:          generateRandomByteReader(0),
			blockLinkSize: 0,
			shouldFail:    true,
			err:           ErrBlockDataEmpty,
		},
		{
			name:          "",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    true,
			err:           ErrBlockNameEmpty,
		},
		{
			name:          " ",
			data:          generateRandomByteReader(3),
			blockLinkSize: 0,
			shouldFail:    true,
			err:           ErrBlockNameEmpty,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			digest, createErr := ps.CreateBlock(ctx, tc.name, tc.data)
			if tc.shouldFail {
				require.NotNil(t, createErr)
				require.Equal(t, createErr, tc.err)
			} else {
				require.Nil(t, createErr)
				rootCid, decodeErr := cid.Decode(digest)
				require.Nil(t, decodeErr)

				rootBlock, readErr := ps.GetBlock(ctx, rootCid)
				require.Nil(t, readErr)
				require.Nil(t, rootBlock.Data)
				require.Equal(t, 1, len(rootBlock.Links))

				rootLink := rootBlock.Links[0]
				require.Equal(t, strings.TrimSpace(tc.name), rootLink.Name)
				blockCid, cidErr := cid.Decode(rootLink.Hash)
				require.Nil(t, cidErr)

				block, readErr := ps.GetBlock(ctx, blockCid)
				require.Nil(t, readErr)

				require.Equal(t, tc.blockLinkSize, len(block.Links))
			}

		})
	}
}
