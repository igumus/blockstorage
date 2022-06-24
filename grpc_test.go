package blockstorage

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/igumus/blockstorage/blockpb"
	mockpeer "github.com/igumus/blockstorage/peer/mock"
	"github.com/igumus/go-objectstore-lib"
	"github.com/igumus/go-objectstore-lib/mock"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

func (s *blockStorageSuite) TestBlockCreationViaGrpc() {
	ctx := context.Background()
	server, lis, setup, teardown := makeGrpcServer()

	store := mock.NewMockObjectStore(s.ctrl)
	store.EXPECT().HasObject(gomock.Any(), gomock.Any()).AnyTimes().Return(true)
	store.EXPECT().CreateObject(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, r io.Reader) (cid.Cid, error) {
		data, err := ioutil.ReadAll(r)
		require.NoError(s.T(), err)

		return objectstore.DigestPrefix.Sum(data)
	})
	peer := mockpeer.NewMockBlockStoragePeer(s.ctrl)
	peer.EXPECT().AnnounceBlock(gomock.Any(), gomock.Any()).AnyTimes().Return(true)

	_, err := newBlockStorage(ctx, EnableDebugMode(), WithLocalStore(store), WithPeer(peer), EnableGrpcEndpoint(server))
	require.NoError(s.T(), err)

	bufDialer := bufDialerFunc(lis)
	go setup()
	defer teardown()

	testCases := []struct {
		name string
		data io.Reader
		code codes.Code
	}{
		{
			name: "valid_name_valid_data",
			data: generateRandomByteReader(s.T(), 3),
			code: codes.OK,
		},
		{
			name: " spaced_name ",
			data: generateRandomByteReader(s.T(), 3),
			code: codes.OK,
		},
		{
			name: "valid_name_empty_data",
			data: generateRandomByteReader(s.T(), 0),
			code: codes.Internal,
		},
		{
			name: "",
			data: generateRandomByteReader(s.T(), 3),
			code: codes.InvalidArgument,
		},
		{
			name: " ",
			data: generateRandomByteReader(s.T(), 3),
			code: codes.InvalidArgument,
		},
		{
			name: "           ",
			data: generateRandomByteReader(s.T(), 3),
			code: codes.InvalidArgument,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		s.T().Run(tc.name, func(t *testing.T) {
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}
			defer conn.Close()
			client := blockpb.NewBlockStorageGrpcServiceClient(conn)
			stream, streamErr := client.WriteBlock(ctx)
			if streamErr != nil {
				log.Fatalf("testingErr: creating stream failed: %s\n", streamErr.Error())
			}
			require.Nil(t, streamErr)
			_, err = toGrpcStream(tc.name, tc.data, stream)
			if tc.code != codes.OK {
				require.NotNil(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.code, st.Code())
			} else {
				require.Nil(t, err)
			}
		})

	}

}
