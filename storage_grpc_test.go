package blockstorage_test

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/igumus/blockstorage"
	"github.com/igumus/blockstorage/blockpb"
	fsstore "github.com/igumus/go-objectstore-fs"
	"github.com/igumus/go-objectstore-lib"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const dataDir = "/tmp"
const dataBucket = "peer"
const bufSize = 1024 * 1024
const chunkSize = 512 << 10

var lis *bufconn.Listener
var store objectstore.ObjectStore

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	store, _ := fsstore.NewFileSystemObjectStore(fsstore.WithDataDir(dataDir), fsstore.WithBucket(dataBucket))
	storage, _ := blockstorage.NewBlockStorage(context.Background(), blockstorage.WithObjectStore(store))
	blockpb.RegisterBlockStorageGrpcServiceServer(s, storage)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func fileToGrpcStream(filePath string, reader io.Reader, stream blockpb.BlockStorageGrpcService_WriteBlockClient) error {
	filename := filepath.Base(filepath.Clean(filePath))

	if sendErr := stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_Name{
			Name: filename,
		},
	}); sendErr != nil {
		return sendErr
	}
	var buf []byte
	const chunkSize = 512 << 10
	for {
		buf = make([]byte, chunkSize)
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}

		sendErr := stream.Send(&blockpb.WriteBlockRequest{
			Data: &blockpb.WriteBlockRequest_ChunkData{
				ChunkData: buf[:n],
			},
		})
		if sendErr != nil {
			return sendErr
		}
	}
	return nil
}

func TestBlockCreation(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := blockpb.NewBlockStorageGrpcServiceClient(conn)
	stream, streamErr := client.WriteBlock(ctx)
	require.Nil(t, streamErr)

	filePath := "/Users/igumus/Desktop/TIFF/sample_1.tiff"
	filename := filepath.Base(filepath.Clean(filePath))

	sendErr := stream.Send(&blockpb.WriteBlockRequest{
		Data: &blockpb.WriteBlockRequest_Name{
			Name: filename,
		},
	})
	require.Nil(t, sendErr)

	reader, err := os.Open(filePath)
	stat, _ := reader.Stat()
	log.Printf("file size: %d\n", stat.Size())
	require.Nil(t, err)
	defer reader.Close()
	var buf []byte
	const chunkSize = 256 << 10
	for {
		buf = make([]byte, chunkSize)
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}

		sendErr := stream.Send(&blockpb.WriteBlockRequest{
			Data: &blockpb.WriteBlockRequest_ChunkData{
				ChunkData: buf[:n],
			},
		})
		require.Nil(t, sendErr)
	}

	ret, err := stream.CloseAndRecv()
	require.Nil(t, err)
	log.Println(ret.Cid)
	require.NotEmpty(t, ret.Cid)

	block, err := client.GetBlock(ctx, &blockpb.GetBlockRequest{Cid: ret.Cid})
	require.Nil(t, err)
	require.Equal(t, len(block.Links), 1)
	require.Equal(t, filename, block.Links[0].Name)
	require.Equal(t, block.Links[0].Tsize, uint64(stat.Size()))

	subblock, err := client.GetBlock(ctx, &blockpb.GetBlockRequest{Cid: block.Links[0].Hash})
	require.Nil(t, err)
	for _, link := range subblock.Links {
		log.Printf("%s, %d\n", link.Hash, link.Tsize)
	}

	// Test for output here.
}
