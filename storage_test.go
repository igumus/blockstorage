package blockstorage_test

import (
	"context"
	"testing"

	"github.com/igumus/blockstorage"
	"github.com/stretchr/testify/require"
)

func TestBlockStorageCreation(t *testing.T) {
	_, storageErr := blockstorage.NewBlockStorage(context.Background())
	require.NotNil(t, storageErr)
}
