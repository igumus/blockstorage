# Block Storage Example
A custom block storage example to manage (store/index etc.) block (aka file/document content) operations in a p2p manner.

## Introduction

`blockstorage` is a p2p service that I wrote to understand the internal workings of block management. Important topics that I focusing on:

1) [Libp2p](https://libp2p.io/)
    - [Kademlia DHT](https://github.com/libp2p/go-libp2p-kad-dht)
        - Content Routing usage
        - Peer Routing usage
        - P2P Protocol usage
    - Peer Discovery (Boostrap Peer, Rendezvous, etc.)
2) [IPLD](https://ipld.io/)
    - Concepts & Spec
    - Proof of Concept custom implementation (via protobuf)
3) [Formats](https://github.com/multiformats/)
    - Multi Address
    - [CID (aka content identifier)](https://docs.ipfs.io/concepts/content-addressing/)



## Libraries/Implementations

* [go-objectstore-lib](https://github.com/igumus/go-objectstore-lib) : Contains base abstraction of storing encoded/serialized objects

* [go-objectstore-fs](https://github.com/igumus/go-objectstore-fs) : Contains file system based functionality to store encoded/serialized objects

## Note:
    - [ ] add `blockstorage` implementation details