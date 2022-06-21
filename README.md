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

* [go-objectstore-fs](https://github.com/igumus/go-objectstore-fs) : Contains file system based implementation/functionality to store encoded/serialized objects

## Layout

- [api/proto](./api/protobuf/) : Contains protobuf definitions
- [blockpb](./blockpb) : Contains generated go source files according to [store.proto](./api/protobuf/store.proto)
- [storage_error.go](./storage_error.go) : Contains `blockstorage` error definitions and error checking functions
- [storage_grpc.go](./storage_grpc.go) : Contains `blockstorage` GRPC endpoint definition and RPC function implementations
- [storage_impl.go](./storage_impl.go) : Contains `BlockStorage` interface implementation and helper functions
- [storage_options.go](./storage_options.go) : Contains `BlockStorage` construction option definitions
- [storage_peer.go](./storage_peer.go) : Contains p2p related protocol definition and functions
- [storage.go](./storage.go) : Contains `blockstorage` construction and  `BlockStorage` interface definition

## Status
`blockstorage` is still in progress.

## TODOs
- [ ] add block indexing mechanism
- [ ] add garbage collection trigger mechanism (to temporaryStore)
- [ ] add long term storage trigger
- [ ] ...