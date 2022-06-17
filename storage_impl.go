package blockstorage

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/ipfs/go-cid"
	"google.golang.org/protobuf/proto"
)

func (s *storage) getBlock(ctx context.Context, cid cid.Cid) (*blockpb.Block, error) {
	data, err := s.store.ReadObject(ctx, cid)
	if err != nil {
		return nil, err
	}

	var block blockpb.Block
	if err := proto.Unmarshal(data, &block); err != nil {
		return nil, err
	}
	return &block, nil
}

// persistBlock - persists given block instance to underlying objectstore. Successful persistence
// returns link to persisted block. Otherwise returns nil and error
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

// createBlock - creates block with given `name` in underlying objectstore. Reads `chunkSize` (default: 512KB) of data
// from reader and constructs a DAG (directed acyclic graph).
func (s *storage) createBlock(ctx context.Context, name string, reader io.Reader) (string, error) {
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
			} else {
				if n > 0 {
					log.Println("dosya bitti kalan byte'lari yaziyorum")
					link, linkErr := s.persistBlockWithData(ctx, buf[:])
					if linkErr != nil {
						return "", linkErr
					}

					links = append(links, link)
					totalSize += uint64(n)
				}
				break
			}
		}

		if n > 0 {
			link, linkErr := s.persistBlockWithData(ctx, buf[:n])
			if linkErr != nil {
				return "", linkErr
			}

			links = append(links, link)
			totalSize += uint64(n)
		} else {
			log.Println("read 0 bytes")
		}

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
