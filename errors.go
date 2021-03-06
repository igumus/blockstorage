package blockstorage

import (
	"errors"
)

// ErrBlockNameEmpty is return, when persisting new block name is empty.
var ErrBlockNameEmpty = errors.New("blockstorage: block name should not be empty")

// ErrBlockDataEmpty is return, when persisting new block has no data
var ErrBlockDataEmpty = errors.New("blockstorage: block data should not be empty")

// ErrBlockIdentifierNotValid is return, when block cid (aka content identifier) not valid
var ErrBlockIdentifierNotValid = errors.New("blockstorage: block identifier not valid")

// ErrBlockProviderNotFound is return, when there is no owner of specified block.
var ErrBlockProviderNotFound = errors.New("blockstorage: not found any provider for block")
