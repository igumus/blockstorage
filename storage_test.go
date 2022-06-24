package blockstorage

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type blockStorageSuite struct {
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
func TestBlockStorageSuite(t *testing.T) {
	suite.Run(t, new(blockStorageSuite))
}

func (s *blockStorageSuite) SetupTest() {
	s.Assertions = require.New(s.T())
	s.ctrl = gomock.NewController(s.T())
}

func (s *blockStorageSuite) TearDownTest() {
	s.ctrl.Finish()
}
