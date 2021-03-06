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
- [blockpb](./blockpb/) : Contains protobuf and grpc related objects according to [store.proto](./api/protobuf/store.proto)
- [blockpb/store_aux.go](./blockpb/store_aux.go) : Contains auxiliary functions/definitions to extends proto objects
- [util/ctx.go](./util/ctx.go) : Contains context cheking helper function and error definitions
- [peer](./peer/) : Contains p2p functions and definitions.
- [errors.go](./errors.go) : Contains `blockstorage` error definitions and error checking functions
- [grpc](./grpc/) : Contains `blockstorage` GRPC endpoint definition and RPC function implementations
- [impl.go](./impl.go) : Contains `BlockStorage` interface implementation and helper functions
- [options.go](./options.go) : Contains `BlockStorage` construction option definitions
- [peer.go](./peer.go) : Contains p2p related protocol definition and functions
- [storage.go](./storage.go) : Contains `blockstorage` construction and  `BlockStorage` interface definition

## Status
`blockstorage` is still in progress.

## TODOs
- Prepare Step
    - [x] Project Structure
    - [x] Makefile tasks
- Grpc Step
    - [x] proto file
    - [x] generate objects
    - [x] add makefile task
    - [x] implement service endpoint
- Peer Step
    - [x] protocol definition
    - [x] store definition 
- Future
    - [ ] add block indexing mechanism
    - [ ] add garbage collection trigger mechanism (to temporaryStore)
    - [ ] add long term storage trigger
