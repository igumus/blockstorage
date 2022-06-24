package blockstorage

import (
	"context"
	"testing"

	"github.com/igumus/blockstorage/util"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestOneNetworkPeerContextHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p1s, p1shutdown, p1err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	defer p1shutdown()

	cid, err := cid.Decode(notExistsCid)
	require.Nil(t, err)

	cancel()
	_, err = p1s.findBlockProvider(ctx, cid)
	require.Equal(t, util.ErrOperationCancelled, err)

}

func TestOneNetworkPeerNotExistsBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p1s, p1shutdown, p1err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	defer p1shutdown()

	cid, err := cid.Decode(notExistsCid)
	require.Nil(t, err)

	require.Equal(t, false, p1s.localStore.HasObject(ctx, cid))

	_, err = p1s.findBlockProvider(ctx, cid)
	require.Equal(t, ErrBlockProviderNotFound, err)
}

func TestOneNetworkPeerBlockCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p1s, p1shutdown, p1err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	defer p1shutdown()

	fileName := "sample.file"
	totalSize := 512 << 10

	digest, creationErr := p1s.CreateBlock(ctx, fileName, generateRandomByteReader(totalSize))
	require.Nil(t, creationErr)
	require.NotEmpty(t, digest)

	cid, decodeErr := cid.Decode(digest)
	require.Nil(t, decodeErr)

	require.True(t, p1s.localStore.HasObject(ctx, cid))

	providers, err := p1s.findBlockProvider(ctx, cid)
	require.Nil(t, err)
	require.Equal(t, 1, len(providers))

	provider := providers[0]
	require.Equal(t, p1s.host.ID(), provider.ID)

	block, err := p1s.GetBlock(ctx, cid)
	require.Nil(t, err)
	require.Nil(t, block.Data)
	require.Equal(t, 1, len(block.Links))
	require.Equal(t, fileName, block.Name)
	// require.Equal(t, uint64(blockblock link.Tsize)
}

func TestTwoNetworkPeersBlockFetching(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p1s, p1shutdown, p1err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String())
	require.Nil(t, p1err)
	require.NotNil(t, p1s)
	defer p1shutdown()

	p2s, p2shutdown, p2err := makeStoragePeer(ctx, 2, bootstrapHost.ID().String())
	require.Nil(t, p2err)
	require.NotNil(t, p2s)
	defer p2shutdown()

	fileName := "sample.file"
	totalSize := 512 << 10

	digest, creationErr := p1s.CreateBlock(ctx, fileName, generateRandomByteReader(totalSize))
	require.Nil(t, creationErr)
	require.NotEmpty(t, digest)

	rootcid, decodeErr := cid.Decode(digest)
	require.Nil(t, decodeErr)

	require.True(t, p1s.localStore.HasObject(ctx, rootcid))
	require.False(t, p1s.tempStore.HasObject(ctx, rootcid))

	providers, err := p1s.findBlockProvider(ctx, rootcid)
	require.Nil(t, err)
	require.Equal(t, 1, len(providers))

	provider := providers[0]
	require.Equal(t, p1s.host.ID(), provider.ID)

	block, err := p1s.GetBlock(ctx, rootcid)
	require.Nil(t, err)
	require.Nil(t, block.Data)
	require.Equal(t, 1, len(block.Links))
	require.Equal(t, fileName, block.Name)

	require.False(t, p2s.localStore.HasObject(ctx, rootcid))
	remoteBlock, remoteErr := p2s.GetBlock(ctx, rootcid)
	require.Nil(t, remoteErr)
	require.True(t, p2s.tempStore.HasObject(ctx, rootcid))

	require.Nil(t, remoteBlock.Data)
	require.Equal(t, len(block.Links), len(remoteBlock.Links))
	require.Equal(t, remoteBlock.Name, block.Name)

	childBlockDigest := remoteBlock.Links[0].Hash
	childCid, err := cid.Decode(childBlockDigest)
	require.Nil(t, err)
	require.True(t, p2s.tempStore.HasObject(ctx, childCid))

}
