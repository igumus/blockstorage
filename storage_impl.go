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

// GetBlock - reads block with given cid (aka content identifier) from underlying object store
func (s *storage) GetBlock(ctx context.Context, cid cid.Cid) (*blockpb.Block, error) {
	var data []byte
	var err error

	if s.store.HasObject(ctx, cid) {
		data, err = s.store.ReadObject(ctx, cid)
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

// persistBlock - persists given block instance to underlying objectstore.
// When persistence success, try to announce block ownership to the network and returns link to persisted block.
// Unsuccessful persistence returns cause error
func (s *storage) persistBlock(ctx context.Context, block *blockpb.Block) (*blockpb.Link, error) {
	blockBin, blockErr := proto.Marshal(block)
	if blockErr != nil {
		return nil, blockErr
	}

	digest, persistErr := s.store.CreateObject(ctx, bytes.NewReader(blockBin))
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

// CreateBlock - creates block with given `name` in underlying objectstore. Reads `chunkSize` (default: 512KB) of data
// from reader and constructs a DAG (directed acyclic graph).
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

	internalLink, internalLinkErr := s.persistBlockWithLinks(ctx, links...)
	if internalLinkErr != nil {
		return "", internalLinkErr
	}
	internalLink.Name = name
	internalLink.Tsize = totalSize

	link, linkErr := s.persistBlockWithLinks(ctx, internalLink)
	if linkErr != nil {
		return "", linkErr
	}

	return link.Hash, nil
}
