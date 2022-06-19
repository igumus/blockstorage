package blockstorage_test

import (
	"context"
	"io"
	"log"
	"net"
	"testing"

	"github.com/igumus/blockstorage"
	"github.com/igumus/blockstorage/blockpb"
	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	store, _ := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir(dataDir), fsstore.WithBucket(dataBucket))
	tstore, _ := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir(dataDir), fsstore.WithBucket(dataBucket+"-temp"))
	blockstorage.NewBlockStorage(context.Background(), blockstorage.WithLocalStore(store), blockstorage.EnableGrpcEndpoint(s), blockstorage.WithTempStore(tstore))
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func toGrpcStream(filename string, reader io.Reader, stream blockpb.BlockStorageGrpcService_WriteBlockClient) (string, error) {
	if sendErr := stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_Name{
			Name: filename,
		},
	}); sendErr != nil {
		return "", sendErr
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
			return "", sendErr
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
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}
			defer conn.Close()
			client := blockpb.NewBlockStorageGrpcServiceClient(conn)
			stream, streamErr := client.WriteBlock(ctx)
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
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := blockpb.NewBlockStorageGrpcServiceClient(conn)
	cancel()
	_, streamErr := client.WriteBlock(ctx)
	require.NotNil(t, streamErr)
}

func TestGrpcContextCancellationAfterStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := blockpb.NewBlockStorageGrpcServiceClient(conn)
	stream, streamErr := client.WriteBlock(ctx)
	require.Nil(t, streamErr)

	sendErr := stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_Name{
			Name: "filename",
		},
	})
	require.Nil(t, sendErr)
	cancel()

	sendErr = stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_ChunkData{
			ChunkData: []byte("selam"),
		},
	})
	if sendErr != nil {
		log.Println("testDebug: EOF error occurred")
		require.Equal(t, io.EOF, sendErr)
	}

	resp, err := stream.CloseAndRecv()
	require.Error(t, err)
	require.Nil(t, resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Canceled, st.Code())
}
