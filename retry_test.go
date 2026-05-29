package upload

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryManager_execute_success(t *testing.T) {
	rm := newRetryManager(DefaultRetryConfig())
	calls := 0
	err := rm.execute(context.Background(), func(context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryManager_execute_retries(t *testing.T) {
	rm := newRetryManager(RetryConfig{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, Multiplier: 1})
	calls := 0
	err := rm.execute(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("fail")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRetryManager_execute_allFail(t *testing.T) {
	rm := newRetryManager(RetryConfig{MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, Multiplier: 1})
	err := rm.execute(context.Background(), func(context.Context) error { return errors.New("always") })
	if err == nil {
		t.Fatal()
	}
}

func TestRetryManager_execute_context(t *testing.T) {
	rm := newRetryManager(DefaultRetryConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rm.execute(ctx, func(context.Context) error { return errors.New("x") })
	if err != context.Canceled {
		t.Fatal(err)
	}
}

func TestRetryManager_jitter(t *testing.T) {
	rm := newRetryManager(DefaultRetryConfig())
	d := rm.jitter(time.Second)
	if d <= 0 {
		t.Fatal(d)
	}
}

func TestNormalizeRetry(t *testing.T) {
	r := RetryConfig{}
	normalizeRetry(&r)
	if r.MaxAttempts != 3 {
		t.Fatal(r)
	}
	r2 := RetryConfig{MaxAttempts: 1, InitialDelay: 0, MaxDelay: 0, Multiplier: 0}
	normalizeRetry(&r2)
	if r2.InitialDelay <= 0 || r2.Multiplier <= 0 {
		t.Fatal(r2)
	}
}
