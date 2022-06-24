package grpc

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/igumus/blockstorage"
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
)

func (s *grpcSuite) TestBlockCreationViaGrpc() {
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

	storage, err := blockstorage.NewFakeBlockStorage(ctx,
		blockstorage.EnableDebugMode(),
		blockstorage.WithLocalStore(store),
		blockstorage.WithPeer(peer),
		blockstorage.EnableGrpcEndpoint(server),
	)
	require.NoError(s.T(), err)

	endpoint, err := NewBlockStorageServiceEndpoint(ctx, storage)
	require.NoError(s.T(), err)
	blockpb.RegisterBlockStorageGrpcServiceServer(server, endpoint)

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
