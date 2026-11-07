package upload

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrInvalidConfig  = errors.New("upload: invalid config")
	ErrNotInitialized = errors.New("upload: not initialized")
	ErrNotFound       = errors.New("upload: object not found")
	ErrUnsupported    = errors.New("upload: unsupported operation")
	ErrObjectTooLarge = errors.New("upload: object too large for simple upload")
	// ErrInvalidRange is returned for negative offsets, non-positive lengths, or offset+length overflow.
	ErrInvalidRange = errors.New("upload: invalid range")
)

// abortTimeout bounds best-effort multipart abort calls.
const abortTimeout = 10 * time.Second

// validateRange checks offset and length and returns the inclusive end offset
// (offset+length-1) without overflowing int64.
func validateRange(offset, length int64) (int64, error) {
	if offset < 0 || length <= 0 {
		return 0, fmt.Errorf("%w: offset=%d length=%d", ErrInvalidRange, offset, length)
	}
	if offset > math.MaxInt64-length {
		return 0, fmt.Errorf("%w: offset+length overflows int64 (offset=%d length=%d)", ErrInvalidRange, offset, length)
	}
	return offset + length - 1, nil
}

// abortContext returns a context for abort calls that survives cancellation of ctx.
func abortContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), abortTimeout)
}

// joinAbortErr returns err joined with abortErr when the abort itself failed.
func joinAbortErr(err, abortErr error) error {
	if abortErr == nil {
		return err
	}
	return errors.Join(err, fmt.Errorf("upload: abort multipart: %w", abortErr))
}
