package grpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"log"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/igumus/blockstorage/blockpb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func makeGrpcServer() (*grpc.Server, *bufconn.Listener, func(), func()) {
	s := grpc.NewServer()
	lis := bufconn.Listen(bufSize)
	return s, lis, func() {
			if err := s.Serve(lis); err != nil {
				log.Fatalf("Server exited with error: %v", err)
			}
		}, func() {
			s.GracefulStop()
		}
}

func bufDialerFunc(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
}

func toGrpcStream(filename string, reader io.Reader, stream blockpb.BlockStorageGrpcService_WriteBlockClient) (string, error) {
	if sendErr := stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_Name{
			Name: filename,
		},
	}); sendErr != nil {
		return "", stream.RecvMsg(nil)
	}

	var buf []byte
	for {
		buf = make([]byte, 512<<10)
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}

		sendErr := stream.Send(&blockpb.WriteBlockRequest{
			Data: &blockpb.WriteBlockRequest_ChunkData{
				ChunkData: buf[:n],
			},
		})
		if sendErr != nil {
			return "", stream.RecvMsg(nil)
		}
	}

	resp, respErr := stream.CloseAndRecv()
	if respErr != nil {
		return "", respErr
	}

	return resp.Cid, nil
}

type grpcSuite struct {
	suite.Suite
	*require.Assertions
	ctrl *gomock.Controller
}

func generateRandomByteReader(t *testing.T, size int) io.Reader {
	if size == 0 {
		return bytes.NewReader([]byte{})
	}
	blk := make([]byte, size)
	_, err := rand.Read(blk)
	require.NoError(t, err)
	return bytes.NewReader(blk)

}
func TestGrpcSuite(t *testing.T) {
	suite.Run(t, new(grpcSuite))
}

func (s *grpcSuite) SetupTest() {
	s.Assertions = require.New(s.T())
	s.ctrl = gomock.NewController(s.T())
}

func (s *grpcSuite) TearDownTest() {
	s.ctrl.Finish()
}
