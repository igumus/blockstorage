package blockstorage

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/ipfs/go-cid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *storage) registerGrpc(server *grpc.Server) {
	if server != nil {
		blockpb.RegisterBlockStorageGrpcServiceServer(server, &storageGrpc{storage: s})
		log.Println("info: grpc endpoint registration success")
	} else {
		log.Println("info: grpc endpoint registration skipped")
	}
}

type storageGrpc struct {
	blockpb.UnimplementedBlockStorageGrpcServiceServer
	storage BlockStorage
}

func (s *storageGrpc) rpcError(code codes.Code, err error) error {
	return status.Error(code, err.Error())
}

func (s *storageGrpc) GetBlock(ctx context.Context, req *blockpb.GetBlockRequest) (*blockpb.Block, error) {
	ctxErr := s.storage.checkContext(ctx)
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

func (s *storageGrpc) WriteBlock(stream blockpb.BlockStorageGrpcService_WriteBlockServer) error {
	ctx := stream.Context()
	ctxErr := s.storage.checkContext(ctx)
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
			ctxErr := s.storage.checkContext(ctx)
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
