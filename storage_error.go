package blockstorage

import "errors"

// ErrObjectstoreNotDefined is return, when objectstore not specified while constructing `BlockStorage` service
var ErrObjectstoreNotDefined = errors.New("blockstorage: objectstore instance not specified")

// ErrBlockNameEmpty is return, when persisting new block name is empty.
var ErrBlockNameEmpty = errors.New("blockstorage: block name should not be empty")

// ErrBlockDataEmpty is return, when persisting new block has no data
var ErrBlockDataEmpty = errors.New("blockstorage: block data should not be empty")

// ErrFindBlockProviderCancelled is return, when finding block provider cancelled via context
var ErrFindBlockProviderCancelled = errors.New("blockstorage: finding block provider cancelled")

// ErrFindBlockProviderTimedOut is return, when finding block provider deadline exceeded
var ErrFindBlockProviderTimedOut = errors.New("blockstorage: finding block provider timed out")

// ErrBlockProviderNotFound is return, when there is no owner of specified block.
var ErrBlockProviderNotFound = errors.New("blockstorage: not found any provider for block")
