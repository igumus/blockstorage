// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package blockpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BlockStorageGrpcServiceClient is the client API for BlockStorageGrpcService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlockStorageGrpcServiceClient interface {
	WriteBlock(ctx context.Context, opts ...grpc.CallOption) (BlockStorageGrpcService_WriteBlockClient, error)
}

type blockStorageGrpcServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockStorageGrpcServiceClient(cc grpc.ClientConnInterface) BlockStorageGrpcServiceClient {
	return &blockStorageGrpcServiceClient{cc}
}

func (c *blockStorageGrpcServiceClient) WriteBlock(ctx context.Context, opts ...grpc.CallOption) (BlockStorageGrpcService_WriteBlockClient, error) {
	stream, err := c.cc.NewStream(ctx, &BlockStorageGrpcService_ServiceDesc.Streams[0], "/blockpb.BlockStorageGrpcService/WriteBlock", opts...)
	if err != nil {
		return nil, err
	}
	x := &blockStorageGrpcServiceWriteBlockClient{stream}
	return x, nil
}

type BlockStorageGrpcService_WriteBlockClient interface {
	Send(*WriteBlockRequest) error
	CloseAndRecv() (*WriteBlockResponse, error)
	grpc.ClientStream
}

type blockStorageGrpcServiceWriteBlockClient struct {
	grpc.ClientStream
}

func (x *blockStorageGrpcServiceWriteBlockClient) Send(m *WriteBlockRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *blockStorageGrpcServiceWriteBlockClient) CloseAndRecv() (*WriteBlockResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(WriteBlockResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BlockStorageGrpcServiceServer is the server API for BlockStorageGrpcService service.
// All implementations must embed UnimplementedBlockStorageGrpcServiceServer
// for forward compatibility
type BlockStorageGrpcServiceServer interface {
	WriteBlock(BlockStorageGrpcService_WriteBlockServer) error
	mustEmbedUnimplementedBlockStorageGrpcServiceServer()
}

// UnimplementedBlockStorageGrpcServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBlockStorageGrpcServiceServer struct {
}

func (UnimplementedBlockStorageGrpcServiceServer) WriteBlock(BlockStorageGrpcService_WriteBlockServer) error {
	return status.Errorf(codes.Unimplemented, "method WriteBlock not implemented")
}
func (UnimplementedBlockStorageGrpcServiceServer) mustEmbedUnimplementedBlockStorageGrpcServiceServer() {
}

// UnsafeBlockStorageGrpcServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlockStorageGrpcServiceServer will
// result in compilation errors.
type UnsafeBlockStorageGrpcServiceServer interface {
	mustEmbedUnimplementedBlockStorageGrpcServiceServer()
}

func RegisterBlockStorageGrpcServiceServer(s grpc.ServiceRegistrar, srv BlockStorageGrpcServiceServer) {
	s.RegisterService(&BlockStorageGrpcService_ServiceDesc, srv)
}

func _BlockStorageGrpcService_WriteBlock_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BlockStorageGrpcServiceServer).WriteBlock(&blockStorageGrpcServiceWriteBlockServer{stream})
}

type BlockStorageGrpcService_WriteBlockServer interface {
	SendAndClose(*WriteBlockResponse) error
	Recv() (*WriteBlockRequest, error)
	grpc.ServerStream
}

type blockStorageGrpcServiceWriteBlockServer struct {
	grpc.ServerStream
}

func (x *blockStorageGrpcServiceWriteBlockServer) SendAndClose(m *WriteBlockResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *blockStorageGrpcServiceWriteBlockServer) Recv() (*WriteBlockRequest, error) {
	m := new(WriteBlockRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BlockStorageGrpcService_ServiceDesc is the grpc.ServiceDesc for BlockStorageGrpcService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlockStorageGrpcService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "blockpb.BlockStorageGrpcService",
	HandlerType: (*BlockStorageGrpcServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WriteBlock",
			Handler:       _BlockStorageGrpcService_WriteBlock_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "store.proto",
}
