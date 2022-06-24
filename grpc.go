package blockstorage

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/igumus/blockstorage/util"
	"github.com/ipfs/go-cid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// registerGrpc - registers `BlockStorageGrpcService` server endpoint
// to given grpc server instance if `EnableGrpcEndpoint` config option
// specified while constructing `BlockStorage` instance. Otherwise just logs
// server endpoint registration skipped
func (s *storage) registerGrpc(server *grpc.Server) {
	if server != nil {
		blockpb.RegisterBlockStorageGrpcServiceServer(server, &storageGrpc{storage: s})
		log.Println("info: grpc endpoint registration success")
	} else {
		log.Println("info: grpc endpoint registration skipped")
	}
}

// Captures/Respresents grpc server endpoint information
type storageGrpc struct {
	blockpb.UnimplementedBlockStorageGrpcServiceServer
	storage BlockStorage
}

// rpcErr - converts given error and grpc error code to grpc status error
func (s *storageGrpc) rpcError(code codes.Code, err error) error {
	return status.Error(code, err.Error())
}

// GetBlock - is a RPC function defined in `store.proto` file. Accepts `blockpb.GetBlockRequest` which contains
// block cid as string. After decoding block cid string to actual cid, asks to underlying `BlockStorage` instance
// to get block
func (s *storageGrpc) GetBlock(ctx context.Context, req *blockpb.GetBlockRequest) (*blockpb.Block, error) {
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return nil, s.rpcError(codes.Aborted, ctxErr)
	}
	digest := req.GetCid()
	cid, decodeErr := cid.Decode(digest)
	if decodeErr != nil {
		return nil, s.rpcError(codes.InvalidArgument, ErrBlockIdentifierNotValid)
	}

	return s.storage.GetBlock(ctx, cid)
}

// WriteBlock - is a rpc function defined in `store.proto` file. Accepts client stream which contains
// document name and raw chunks of document content and writes to permanent object store.
//
// On successful function call, returns `nil` with code `codes.OK`. Otherwise;
// - On context error: returns associated context error with code `codes.Aborted`
// - On receive error: returns associated error with code `codes.Aborted`
// - On empty document name err: returns `ErrBlockNameEmpty` error with code `codes.InvalidArgument`
// - On other errors: returns associated error with code `codes.Internal`
func (s *storageGrpc) WriteBlock(stream blockpb.BlockStorageGrpcService_WriteBlockServer) error {
	ctx := stream.Context()
	ctxErr := util.CheckContext(ctx)
	if ctxErr != nil {
		return s.rpcError(codes.Aborted, ctxErr)
	}
	request, requestErr := stream.Recv()
	if requestErr != nil {
		log.Printf("err: receiving request failed: %s\n", requestErr.Error())
		return status.Error(codes.Aborted, "cannot receive request")
	}

	fname := request.GetName()
	fileName := strings.TrimSpace(fname)
	if fileName == "" {
		return s.rpcError(codes.InvalidArgument, ErrBlockNameEmpty)
	}

	pr, pw := io.Pipe()
	go func() {
		var retErr error = nil
		for {
			ctxErr := util.CheckContext(ctx)
			if ctxErr != nil {
				retErr = ctxErr
			}
			req, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("err: receiving chunk failed: %v\n", err)
				retErr = err
				break
			}
			_, err = pw.Write(req.GetChunkData())
			if err != nil {
				retErr = err
				break
			}
		}
		if retErr != nil {
			pw.CloseWithError(retErr)
		} else {
			pw.Close()
		}

	}()

	digest, err := s.storage.CreateBlock(ctx, fileName, pr)
	if err != nil {
		log.Printf("err: writing block failed: %s, %s\n", fileName, err.Error())
		return s.rpcError(codes.Internal, err)
	}

	return stream.SendAndClose(&blockpb.WriteBlockResponse{
		Cid: digest,
	})
}
