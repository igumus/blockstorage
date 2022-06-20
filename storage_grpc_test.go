package blockstorage

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"testing"

	"github.com/igumus/blockstorage/blockpb"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

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
		buf = make([]byte, chunkSize)
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

func TestBlockCreationViaGrpc(t *testing.T) {
	ctx := context.Background()
	server, lis, setup, teardown := makeGrpcServer()
	bufDialer := bufDialerFunc(lis)
	_, shutdown, err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String(), EnableGrpcEndpoint(server))
	require.NoError(t, err)
	go setup()
	defer teardown()
	defer shutdown()

	testCases := []struct {
		name string
		data io.Reader
		code codes.Code
	}{
		{
			name: "valid_name_valid_data",
			data: generateRandomByteReader(3),
			code: codes.OK,
		},
		{
			name: " spaced_name ",
			data: generateRandomByteReader(3),
			code: codes.OK,
		},
		{
			name: "valid_name_empty_data",
			data: generateRandomByteReader(0),
			code: codes.Internal,
		},
		{
			name: "",
			data: generateRandomByteReader(3),
			code: codes.InvalidArgument,
		},
		{
			name: " ",
			data: generateRandomByteReader(3),
			code: codes.InvalidArgument,
		},
		{
			name: "           ",
			data: generateRandomByteReader(3),
			code: codes.InvalidArgument,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
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
			digest, err := toGrpcStream(tc.name, tc.data, stream)
			if tc.code != codes.OK {
				require.NotNil(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.code, st.Code())
			} else {
				require.Nil(t, err)
				_, decodeErr := cid.Decode(digest)
				require.Nil(t, decodeErr)
			}
		})

	}

}

func TestGrpcContextCancellationBeforeStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server, lis, setup, teardown := makeGrpcServer()
	bufDialer := bufDialerFunc(lis)
	_, shutdown, err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String(), EnableGrpcEndpoint(server))
	require.NoError(t, err)
	go setup()
	defer teardown()
	defer shutdown()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := blockpb.NewBlockStorageGrpcServiceClient(conn)
	cancel()
	_, streamErr := client.WriteBlock(ctx)
	require.NotNil(t, streamErr)
	st, ok := status.FromError(streamErr)
	require.True(t, ok)
	require.Equal(t, codes.Canceled, st.Code())
}

func TestGrpcContextCancellationAfterStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, lis, setup, teardown := makeGrpcServer()
	bufDialer := bufDialerFunc(lis)
	_, shutdown, err := makeStoragePeer(ctx, 1, bootstrapHost.ID().String(), EnableGrpcEndpoint(server))
	require.NoError(t, err)
	go setup()
	defer teardown()
	defer shutdown()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := blockpb.NewBlockStorageGrpcServiceClient(conn)
	stream, streamErr := client.WriteBlock(ctx)
	require.Nil(t, streamErr)
	cancel()

	resp, err := toGrpcStream(" ", bytes.NewReader([]byte("selam")), stream)
	require.Error(t, err)
	require.Equal(t, "", resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Canceled, st.Code())
}
