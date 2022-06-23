package blockpb

import "google.golang.org/protobuf/proto"

func Encode(block *Block) ([]byte, error) {
	blockBin, blockErr := proto.Marshal(block)
	if blockErr != nil {
		return nil, blockErr
	}
	return blockBin, nil
}

func Decode(data []byte) (*Block, error) {
	var block Block
	if err := proto.Unmarshal(data, &block); err != nil {
		return nil, err
	}
	return &block, nil
}
