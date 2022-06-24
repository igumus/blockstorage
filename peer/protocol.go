package peer

import (
	"bufio"
	"context"
	"log"

	"github.com/igumus/go-objectstore-lib"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// BlockReadProtocol - holds libp2p protocol identifier for reading block from remote peer
const BlockReadProtocolID = protocol.ID("/blockstorage/block/read/1.0.0")

type ReadProtocol network.StreamHandler

func generateReadProtocol(store objectstore.ObjectStore) func(network.Stream) {
	return func(stream network.Stream) {
		reader := bufio.NewReader(stream)
		_, cid, err := cid.CidFromReader(reader)
		if err != nil {
			log.Printf("err: decoding cid failed: %s\n", err.Error())
			stream.Reset()
		}

		log.Printf("info: incoming cid is : %s\n", cid)
		data, err := store.ReadObject(context.Background(), cid)
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
}
