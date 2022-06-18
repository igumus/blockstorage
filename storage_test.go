package blockstorage_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/igumus/blockstorage"
	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

const dataDir = "/tmp"
const dataBucket = "peer"
const bufSize = 1024 * 1024
const chunkSize = 512 << 10

func generateRandomByteReader(size int) io.Reader {
	if size == 0 {
		return bytes.NewReader([]byte{})
	}
	blk := make([]byte, size)
	_, err := rand.Read(blk)
	if err != nil {
		log.Printf("err: error occured while generating random bytes: %s\n", err)
		return bytes.NewReader([]byte{})
	}
	return bytes.NewReader(blk)

}

func TestBlockStorageInstanceCreation(t *testing.T) {
	_, storageErr := blockstorage.NewBlockStorage(context.Background())
	require.NotNil(t, storageErr)
}

func TestBlockCreation(t *testing.T) {
	store, storeErr := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir(dataDir), fsstore.WithBucket(dataBucket))
	require.NoError(t, storeErr)

	storage, storageErr := blockstorage.NewBlockStorage(context.Background(), blockstorage.WithObjectStore(store))
	require.NoError(t, storageErr)
	t.Parallel()

	testCases := []struct {
		name          string
		data          io.Reader
		shouldFail    bool
		blockLinkSize int
	}{
		{
			name:          "valid_input",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    false,
		},
		{
			name:          "equal_to_chunk_size",
			data:          generateRandomByteReader(512 << 10),
			blockLinkSize: 1,
			shouldFail:    false,
		},
		{
			name:          "double_chunk_size",
			data:          generateRandomByteReader(1024 << 10),
			blockLinkSize: 2,
			shouldFail:    false,
		},
		{
			name:          " spaced_name ",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    false,
		},
		{
			name:          "empty_data",
			data:          generateRandomByteReader(0),
			blockLinkSize: 0,
			shouldFail:    true,
		},
		{
			name:          "",
			data:          generateRandomByteReader(3),
			blockLinkSize: 1,
			shouldFail:    true,
		},
		{
			name:          " ",
			data:          generateRandomByteReader(3),
			blockLinkSize: 0,
			shouldFail:    true,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			digest, createErr := storage.CreateBlock(ctx, tc.name, tc.data)
			if tc.shouldFail {
				require.NotNil(t, createErr)
			} else {
				require.Nil(t, createErr)
				rootCid, decodeErr := cid.Decode(digest)
				require.Nil(t, decodeErr)

				rootBlock, readErr := storage.GetBlock(ctx, rootCid)
				require.Nil(t, readErr)
				require.Nil(t, rootBlock.Data)
				require.Equal(t, 1, len(rootBlock.Links))

				rootLink := rootBlock.Links[0]
				require.Equal(t, strings.TrimSpace(tc.name), rootLink.Name)
				blockCid, cidErr := cid.Decode(rootLink.Hash)
				require.Nil(t, cidErr)

				block, readErr := storage.GetBlock(ctx, blockCid)
				require.Nil(t, readErr)

				require.Equal(t, tc.blockLinkSize, len(block.Links))
			}

		})
	}
}
