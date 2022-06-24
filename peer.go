package blockstorage

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"sync"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/blockstorage/util"
	"github.com/igumus/go-objectstore-lib"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// registerPeerProtocol - registers blockstorage protocol's to specified peer host
func (s *storage) registerPeerProtocol() {
	if s.host == nil {
		log.Println("warn: peer protocol not set because of empty host")
		return
	}
	s.host.SetStreamHandler(BlockReadProtocol, s.blockReadStreamHandler)
	log.Printf("info: peer protocol registered: %s\n", BlockReadProtocol)
}

// BlockReadProtocol - holds libp2p protocol identifier for reading block from remote peer
const BlockReadProtocol = protocol.ID("/blockstorage/block/read/1.0.0")

// blockReadStreamHandler - block protocol's read stream handler
func (s *storage) blockReadStreamHandler(stream network.Stream) {
	if s.debug {
		log.Println("debug: block read stream opened")
	}

	reader := bufio.NewReader(stream)
	_, cid, err := cid.CidFromReader(reader)
	if err != nil {
		log.Printf("err: decoding cid failed: %s\n", err.Error())
		stream.Reset()
	}

	if s.debug {
		log.Printf("debug: incoming cid is : %s\n", cid)
	}
	data, err := s.localStore.ReadObject(context.Background(), cid)
	if err != nil {
		log.Printf("err: reading block object failed in stream: %s, %s\n", cid, err.Error())
		stream.Reset()
	}

	newCid, err := objectstore.DigestPrefix.Sum(data)
	if err != nil {
		log.Printf("warn: digesting block data failed: %s\n", err.Error())
	} else {
		log.Printf("info: compare digests: %s, %s, %t\n", cid, newCid, newCid.Equals(cid))
	}

	n, err := stream.Write(data)
	if err != nil {
		log.Printf("err: writing block content to stream failed: %s, %s\n", cid, err.Error())
		stream.Reset()
	}

	log.Printf("info: written block content to stream successfully: %d bytes\n", n)
	stream.CloseWrite()
}

// announceBlockOwnership - announces ownership of given cid (aka content identifier) to the p2p network.
// Returns `true` in successful announcement, otherwise `false`
func (s *storage) announceBlockOwnership(ctx context.Context, cid cid.Cid) bool {
	if s.crouter == nil {
		if s.debug {
			log.Printf("warn: announcing block skipped: %s, content router not specified\n", cid)
		}
		return false
	}
	if err := s.crouter.Provide(ctx, cid, true); err != nil {
		log.Printf("warn: announcing block failed: %s, %s\n", cid, err.Error())
		return false
	}
	log.Printf("info: announcing block succeed: %s\n", cid)
	return true
}

// findBlockProvider - searches ownership of given cid (aka content identifier) on the p2p network.
// If found any provider, returns address information of that peer(s).
// Otherwise returns `ErrBlockProviderNotFound` error.
func (s *storage) findBlockProvider(ctx context.Context, cid cid.Cid) ([]peer.AddrInfo, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	log.Printf("info: asking object owner to network: %s\n", cid)
	peerCapacity := 3
	chProviders := s.crouter.FindProvidersAsync(ctx, cid, peerCapacity)
	providers := make([]peer.AddrInfo, 0, peerCapacity)
	for provider := range chProviders {
		providers = append(providers, provider)
	}

	if len(providers) > 0 {
		return providers, nil
	}

	return nil, ErrBlockProviderNotFound
}

// fetchRemoteBlock - fetches given cid (aka content identifier) from remote peer
// While fetching creates 1:1 stream with the remote peer.
// On succesful communication returns, byte content of desired block, otherwise returns cause error
func (s *storage) fetchRemoteBlock(ctx context.Context, cid cid.Cid, remotePeerAddr peer.AddrInfo) ([]byte, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	log.Printf("info: fetching object %s from %s\n", cid, remotePeerAddr.ID)
	stream, err := s.host.NewStream(ctx, remotePeerAddr.ID, BlockReadProtocol)
	if err != nil {
		log.Printf("err: creating stream failed: %s, %s\n", remotePeerAddr, err.Error())
		return nil, err
	}
	defer stream.Close()

	bin, err := cid.MarshalBinary()
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(bin)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(stream)

	newCid, createErr := s.tempStore.CreateObject(ctx, bytes.NewReader(data))
	if createErr != nil {
		log.Printf("err: storing remote block to temp store failed: %s, %s\n", cid, createErr.Error())
	} else {
		log.Printf("info: requested block:%s, received block: %s\n", cid, newCid)
	}

	return data, err
}

// getRemoteBlock - is a helper function that gets block with given cid (aka content identifier) from p2p network.
//
// Flow:
// 1. Finds provider for given block cid
// 2. Fetches block from found provider (currently first provider) via `/blockstorage/block/read/1.0.0` peer protocol
// 3. Persists fetched block to temporary object store.
// 4. Returns encoded/marshalled block
//
// Error:
// When any of the flow operations fail, returns `nil` with error cause
func (s *storage) getRemoteBlock(ctx context.Context, rootcid cid.Cid) ([]byte, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, ctxErr
	}
	providers, err := s.findBlockProvider(ctx, rootcid)
	if err != nil {
		return nil, err
	}
	provider := providers[0]

	data, err := s.fetchRemoteBlock(ctx, rootcid, provider)
	if err != nil {
		return nil, err
	}

	block, blockErr := blockpb.Decode(data)
	if blockErr != nil {
		log.Printf("err: decoding block failed: %s, %s\n", rootcid, blockErr.Error())
		return data, nil
	}

	if len(block.Links) > 0 {
		wg := sync.WaitGroup{}
		wg.Add(len(block.Links))
		for _, link := range block.Links {
			go func(l *blockpb.Link) {
				defer wg.Done()
				childCid, err := cid.Decode(l.Hash)
				if err != nil {
					log.Printf("err: decoding child cid failed: %s, %s\n", l.Hash, err.Error())
				}
				if _, err := s.fetchRemoteBlock(ctx, childCid, provider); err != nil {
					log.Printf("err: fetching remote object failed: %s, %s\n", childCid, err.Error())
				}
			}(link)
		}
		wg.Wait()
	}

	return data, nil

}
