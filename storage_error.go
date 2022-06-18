package blockstorage

import "errors"

// ErrObjectstoreNotDefined is return, when objectstore not specified while constructing `BlockStorage` service
var ErrObjectstoreNotDefined = errors.New("blockstorage: objectstore instance not specified")

// ErrBlockNameEmpty is return, when persisting new block name is empty.
var ErrBlockNameEmpty = errors.New("blockstorage: block name should not be empty")

// ErrBlockDataEmpty is return, when persisting new block has no data
var ErrBlockDataEmpty = errors.New("blockstorage: block data should not be empty")
