package util

import (
	"context"
	"errors"
)

// ErrOperationCancelled is return, when operation cancelled via context
var ErrOperationCancelled = errors.New("blockstorage: operation context cancelled")

// ErrOperationTimedOut is return, when operation deadline exceeded
var ErrOperationTimedOut = errors.New("blockstorage: operation timed out")

// CheckContext - checks context has error. If context has not err returns nil.
// Otherwise operates following
// - `context.DeadlineExceeded` returns `ErrOperationTimedOut`
// - `context.Canceled` returns `ErrOperationCancelled`
func CheckContext(ctx context.Context) error {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return ErrOperationTimedOut
	case context.Canceled:
		return ErrOperationCancelled
	default:
		return nil
	}
}
