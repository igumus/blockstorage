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
	blockpb.BlockStorageGrpcServiceServer
	storage BlockStorage
}

func (s *storageGrpc) GetBlock(ctx context.Context, req *blockpb.GetBlockRequest) (*blockpb.Block, error) {
	digest := req.GetCid()
	cid, decodeErr := cid.Decode(digest)
	if decodeErr != nil {
		return nil, status.Error(codes.InvalidArgument, "cid is not valid")
	}

	return s.storage.GetBlock(ctx, cid)
}

func (s *storageGrpc) WriteBlock(stream blockpb.BlockStorageGrpcService_WriteBlockServer) error {
	ctx := stream.Context()
	if ctx.Err() != nil {
		log.Printf("err: context failed: %s\n", ctx.Err().Error())
		return status.Error(codes.Canceled, "cancelled via context")
	}
	request, requestErr := stream.Recv()
	if requestErr != nil {
		log.Printf("err: receiving request failed: %s\n", requestErr.Error())
		return status.Error(codes.Aborted, "cannot receive request")
	}

	fname := request.GetName()
	fileName := strings.TrimSpace(fname)
	if fileName == "" {
		return status.Error(codes.InvalidArgument, "name attribute not set")
	}

	pr, pw := io.Pipe()
	go func() {
		var retErr error = nil
		for {
			if ctx.Err() != nil {
				retErr = ctx.Err()
				break
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
		return status.Error(codes.Internal, "writing block failed")
	}

	return stream.SendAndClose(&blockpb.WriteBlockResponse{
		Cid: digest,
	})
}
