package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"testing"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

func TestValidateRange(t *testing.T) {
	end, err := validateRange(0, 10)
	if err != nil || end != 9 {
		t.Fatalf("end=%d err=%v", end, err)
	}
	end, err = validateRange(math.MaxInt64-1, 1)
	if err != nil || end != math.MaxInt64-1 {
		t.Fatalf("end=%d err=%v", end, err)
	}
	for _, tc := range []struct{ off, n int64 }{
		{-1, 1}, {0, 0}, {0, -5},
		{math.MaxInt64, 1}, {1, math.MaxInt64}, {math.MaxInt64 - 5, 10},
	} {
		if _, err := validateRange(tc.off, tc.n); !errors.Is(err, ErrInvalidRange) {
			t.Errorf("validateRange(%d,%d) err=%v", tc.off, tc.n, err)
		}
	}
}

func TestGetRangeOverflowAllBackends(t *testing.T) {
	ctx := context.Background()
	w := &bytes.Buffer{}
	backends := map[string]objectBackend{
		"oss":   &ossBackend{bucket: &mockOSSBucket{objects: map[string][]byte{"k": []byte("abc")}}},
		"cos":   &cosBackend{client: &mockCOS{objects: map[string][]byte{"k": []byte("abc")}}},
		"aws":   &awsBackend{client: &mockAWS{objects: map[string][]byte{"k": []byte("abc")}}, cfg: AWSConfig{Bucket: "b"}},
		"azure": &azureBackend{container: "c"},
		"mem":   func() objectBackend { m := newMemBackend("https://cdn"); m.objects["k"] = []byte("abc"); return m }(),
	}
	for name, b := range backends {
		t.Run(name, func(t *testing.T) {
			if err := b.GetRange(ctx, "k", w, math.MaxInt64-1, 10); !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("overflow err = %v", err)
			}
			if err := b.GetRange(ctx, "k", w, -1, 1); !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("negative offset err = %v", err)
			}
			if err := b.GetRange(ctx, "k", w, 0, 0); !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("zero length err = %v", err)
			}
		})
	}
}

func TestMemBackendGetRangeClampsEnd(t *testing.T) {
	m := newMemBackend("https://cdn")
	m.objects["k"] = []byte("abcdef")
	var buf bytes.Buffer
	if err := m.GetRange(context.Background(), "k", &buf, 2, math.MaxInt64-2); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "cdef" {
		t.Fatalf("got %q", buf.String())
	}
}

// --- abort error joining ---

var errAbort = errors.New("abort failed")

type ossAbortFail struct{ mockOSSBucket }

func (m *ossAbortFail) UploadPart(oss.InitiateMultipartUploadResult, io.Reader, int64, int, ...oss.Option) (oss.UploadPart, error) {
	return oss.UploadPart{}, errors.New("part failed")
}
func (m *ossAbortFail) AbortMultipartUpload(oss.InitiateMultipartUploadResult, ...oss.Option) error {
	return errAbort
}

func TestOSSUploadMultipartAbortErrorJoined(t *testing.T) {
	b := &ossBackend{bucket: &ossAbortFail{}, mp: MultipartConfig{PartSize: 4}}
	err := b.UploadMultipart(context.Background(), "k", bytes.NewReader([]byte("12345678")), 8, b.mp)
	if err == nil || !errors.Is(err, errAbort) {
		t.Fatalf("err = %v, want to wrap abort error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = b.UploadMultipart(ctx, "k", bytes.NewReader([]byte("12345678")), 8, b.mp)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, errAbort) {
		t.Fatalf("canceled err = %v", err)
	}
}

type cosAbortFail struct {
	mockCOS
	abortCtxErr error
}

func (m *cosAbortFail) ObjectAbortMultipartUpload(ctx context.Context, name, uploadID string) (*cos.Response, error) {
	m.abortCtxErr = ctx.Err()
	return nil, errAbort
}

func (m *cosAbortFail) ObjectUploadPart(ctx context.Context, name, uploadID string, partNumber int, r io.Reader, opt *cos.ObjectUploadPartOptions) (*cos.Response, error) {
	return nil, errors.New("part failed")
}

func TestCOSUploadMultipartAbortErrorJoined(t *testing.T) {
	m := &cosAbortFail{}
	b := &cosBackend{client: m, mp: MultipartConfig{PartSize: 4}}
	err := b.UploadMultipart(context.Background(), "k", bytes.NewReader([]byte("12345678")), 8, b.mp)
	if !errors.Is(err, errAbort) {
		t.Fatalf("err = %v", err)
	}

	// Abort must still run with a live context when the caller's ctx is canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = b.UploadMultipart(ctx, "k", bytes.NewReader([]byte("12345678")), 8, b.mp)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, errAbort) {
		t.Fatalf("canceled err = %v", err)
	}
	if m.abortCtxErr != nil {
		t.Fatalf("abort ctx was canceled: %v", m.abortCtxErr)
	}
}

func TestJoinAbortErr(t *testing.T) {
	base := errors.New("base")
	if got := joinAbortErr(base, nil); got != base {
		t.Fatalf("got %v", got)
	}
	got := joinAbortErr(base, errAbort)
	if !errors.Is(got, base) || !errors.Is(got, errAbort) {
		t.Fatalf("got %v", got)
	}
}

// --- retry ---

func TestSleepContext(t *testing.T) {
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if err := sleepContext(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if err := sleepContext(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("zero-delay canceled err = %v", err)
	}
}

func TestRetryManagerCancelDuringSleep(t *testing.T) {
	rm := newRetryManager(RetryConfig{MaxAttempts: 3, InitialDelay: time.Hour, MaxDelay: time.Hour, Multiplier: 1})
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	done := make(chan error, 1)
	go func() {
		done <- rm.execute(ctx, func(context.Context) error {
			calls++
			cancel()
			return errors.New("fail")
		})
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("execute did not honor ctx during backoff")
	}
}

func TestRetryManagerBackoffCaps(t *testing.T) {
	rm := newRetryManager(RetryConfig{MaxAttempts: 4, InitialDelay: 10 * time.Millisecond, MaxDelay: 25 * time.Millisecond, Multiplier: 10})
	var delays []time.Duration
	rm.sleep = func(_ context.Context, d time.Duration) error { delays = append(delays, d); return nil }
	rm.int64n = func(int64) int64 { return 0 }
	_ = rm.execute(context.Background(), func(context.Context) error { return errors.New("x") })
	if len(delays) != 3 {
		t.Fatalf("delays = %v", delays)
	}
	for _, d := range delays {
		if d > 25*time.Millisecond+25*time.Millisecond/5 {
			t.Fatalf("delay %v exceeds cap (delays=%v)", d, delays)
		}
	}
}

func TestRetryJitterBounds(t *testing.T) {
	rm := newRetryManager(DefaultRetryConfig())
	if rm.jitter(0) != 0 {
		t.Fatal("zero delay")
	}
	for i := 0; i < 200; i++ {
		d := rm.jitter(time.Second)
		if d < 800*time.Millisecond || d >= 1200*time.Millisecond {
			t.Fatalf("jitter %v out of [0.8s,1.2s)", d)
		}
	}
	if d := rm.jitter(3); d <= 0 {
		t.Fatalf("tiny delay jitter %v", d)
	}
}

// --- NewStorage ctx ---

func TestNewStorageHonorsContext(t *testing.T) {
	old := buildBackend
	buildBackend = func(cfg Config) (objectBackend, error) {
		return &pingFailBackend{memBackend: newMemBackend("https://cdn"), okAfter: 100}, nil
	}
	defer func() { buildBackend = old }()
	cfg := validTestConfig(ProviderAliyun)
	cfg.Retry = RetryConfig{MaxAttempts: 5, InitialDelay: time.Hour, MaxDelay: time.Hour, Multiplier: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := NewStorage(ctx, cfg)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("NewStorage ignored ctx")
	}
}
