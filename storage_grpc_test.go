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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	store, _ := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir(dataDir), fsstore.WithBucket(dataBucket))
	blockstorage.NewBlockStorage(context.Background(), blockstorage.WithObjectStore(store), blockstorage.EnableGrpcEndpoint(s))
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
		name       string
		data       io.Reader
		shouldFail bool
	}{
		{
			name:       "valid_name_valid_data",
			data:       generateRandomByteReader(3),
			shouldFail: false,
		},
		{
			name:       " spaced_name ",
			data:       generateRandomByteReader(3),
			shouldFail: false,
		},
		{
			name:       "valid_name_empty_data",
			data:       generateRandomByteReader(0),
			shouldFail: true,
		},
		{
			name:       "",
			data:       generateRandomByteReader(3),
			shouldFail: true,
		},
		{
			name:       " ",
			data:       generateRandomByteReader(3),
			shouldFail: true,
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
			if tc.shouldFail {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
				_, decodeErr := cid.Decode(digest)
				require.Nil(t, decodeErr)
			}
		})

	}

}
