package upload

import (
	"context"
	"math/rand/v2"
	"time"
)

type retryManager struct {
	config RetryConfig
	// int64n returns a value in [0, n); defaults to math/rand/v2 (goroutine-safe).
	int64n func(n int64) int64
	sleep  func(ctx context.Context, d time.Duration) error
}

func newRetryManager(cfg RetryConfig) *retryManager {
	return &retryManager{
		config: cfg,
		int64n: rand.Int64N,
		sleep:  sleepContext,
	}
}

// sleepContext waits for d or until ctx is done, releasing the timer either way.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (rm *retryManager) execute(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	delay := rm.config.InitialDelay
	for attempt := 0; attempt < rm.config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := op(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < rm.config.MaxAttempts-1 {
			if err := rm.sleep(ctx, rm.jitter(delay)); err != nil {
				return err
			}
			next := float64(delay) * rm.config.Multiplier
			if next > float64(rm.config.MaxDelay) {
				delay = rm.config.MaxDelay
			} else {
				delay = time.Duration(next)
			}
		}
	}
	return lastErr
}

func (rm *retryManager) jitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	jitterRange := delay / 5
	if jitterRange == 0 {
		jitterRange = 1
	}
	j := time.Duration(rm.int64n(int64(jitterRange * 2)))
	return delay - jitterRange + j
}

func normalizeRetry(r *RetryConfig) {
	if r.MaxAttempts <= 0 {
		*r = DefaultRetryConfig()
	}
	if r.InitialDelay <= 0 {
		r.InitialDelay = DefaultRetryConfig().InitialDelay
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = DefaultRetryConfig().MaxDelay
	}
	if r.Multiplier <= 0 {
		r.Multiplier = DefaultRetryConfig().Multiplier
	}
	if r.MaxDelay < r.InitialDelay {
		r.MaxDelay = r.InitialDelay
	}
}
