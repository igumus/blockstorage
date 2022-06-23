package blockstorage

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/ipfs/go-cid"
	"google.golang.org/protobuf/proto"
)

// GetBlock - reads block with given cid (aka content identifier) from underlying object store.
//
// Flow:
// 1. Finds store that contains block with given cid
// 	1.1. checks permanent object store already has block with given cid, If exists reads from permanent object store.
// 	1.2. checks temporary object store already has block with given cid, If exists reads from temporary object store.
// 	1.3. asks p2p network to provide block with given cid, If founds any provider, stores block to temporary object store.
// 2. Decodes/Unmarshals binary form of block to proto object instance.
// 3. Returns proto instance (`blockpb.Block`) without error
//
// Error:
// When any of the flow operations fail, returns `nil` with error cause
func (s *storage) GetBlock(ctx context.Context, cid cid.Cid) (*blockpb.Block, error) {
	var data []byte
	var err error

	if s.localStore.HasObject(ctx, cid) {
		data, err = s.localStore.ReadObject(ctx, cid)
	} else if s.tempStore.HasObject(ctx, cid) {
		data, err = s.tempStore.ReadObject(ctx, cid)
	} else {
		data, err = s.getRemoteBlock(ctx, cid)
	}
	if err != nil {
		return nil, err
	}

	var block blockpb.Block
	if err := proto.Unmarshal(data, &block); err != nil {
		return nil, err
	}
	return &block, nil
}

// persistBlock - is a helper function that persists given block instance to permanent store.
//
// Flow:
// 1. Encodes/Marshals block proto object instance to binary
// 2. Persists binary content to permanent store.
// 3. Announces block ownership to p2p network.
// 4. Returns proto object instance reference (`blockpb.Link`)
//
// Error:
// When any of the flow operations fail, returns `nil` with error cause
func (s *storage) persistBlock(ctx context.Context, block *blockpb.Block) (*blockpb.Link, error) {
	blockBin, blockErr := proto.Marshal(block)
	if blockErr != nil {
		return nil, blockErr
	}

	digest, persistErr := s.localStore.CreateObject(ctx, bytes.NewReader(blockBin))
	if persistErr != nil {
		return nil, persistErr
	}

	if s.debug {
		log.Printf("debug: wrote block with digest: %s, %d\n", digest.String(), len(block.Data))
	}

	s.announceBlockOwnership(ctx, digest)

	return &blockpb.Link{
		Hash:  digest.String(),
		Tsize: uint64(len(block.Data)),
	}, nil
}

// persistBlockWithData - creates and persists block which only have `Data` field with given byte slice.
func (s *storage) persistBlockWithData(ctx context.Context, data []byte) (*blockpb.Link, error) {
	block := &blockpb.Block{
		Data: data,
	}
	return s.persistBlock(ctx, block)
}

// persistBlockWithLinks - creates and persists block which only have `Links` field with given links
func (s *storage) persistBlockWithLinks(ctx context.Context, links ...*blockpb.Link) (*blockpb.Link, error) {
	block := &blockpb.Block{}
	block.Links = append(block.Links, links...)
	return s.persistBlock(ctx, block)
}

// CreateBlock - creates block with given `name` in underlying objectstore.
//
// Flow:
// 1. Validates file name
// 2. Reads `chunkSize` (default: 512KB) of data from `reader`
//	2.1 On each reading step persists DAG (Directed Acyclic Graph) leaf nodes to permanent store.
// 3. Creates root node to associate with leaf nodes.
// 4. Persists root of DAG to permanent store.
//
// Error:
// - When `fname` is not valid returns `"", ErrBlockNameEmpty`
// - When reading from `reader` fails returns `"", <Reader Failure Error>`
// - When reader not contains any data, returns `"",ErrBlockDataEmpty`
func (s *storage) CreateBlock(ctx context.Context, fname string, reader io.Reader) (string, error) {
	name := strings.TrimSpace(fname)
	if name == "" {
		return "", ErrBlockNameEmpty
	}

	links := make([]*blockpb.Link, 0)
	totalSize := uint64(0)
	var buf []byte
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		buf = make([]byte, s.chunkSize)
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}

		link, linkErr := s.persistBlockWithData(ctx, buf[:n])
		if linkErr != nil {
			return "", linkErr
		}

		links = append(links, link)
		totalSize += uint64(n)
	}

	if len(links) < 1 {
		return "", ErrBlockDataEmpty
	}

	root := &blockpb.Block{
		Name: name,
	}
	root.Links = append(root.Links, links...)

	rootLink, rootLinkErr := s.persistBlock(ctx, root)
	if rootLinkErr != nil {
		return "", rootLinkErr
	}
	return rootLink.Hash, nil
}
