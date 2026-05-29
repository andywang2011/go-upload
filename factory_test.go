package upload

import (
	"context"
	"errors"
	"testing"
)

func TestFactory_unknownProvider(t *testing.T) {
	cfg := validTestConfig(ProviderAliyun)
	cfg.Provider.Name = "nope"
	if _, err := buildBackend(cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewStorage_pingRetry(t *testing.T) {
	old := buildBackend
	buildBackend = func(cfg Config) (objectBackend, error) {
		return &pingFailBackend{memBackend: newMemBackend("https://cdn"), okAfter: 2}, nil
	}
	defer func() { buildBackend = old }()
	cfg := validTestConfig(ProviderAliyun)
	cfg.Retry = RetryConfig{MaxAttempts: 3, InitialDelay: 1, MaxDelay: 1, Multiplier: 1}
	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
}

type pingFailBackend struct {
	*memBackend
	calls   int
	okAfter int
}

func (p *pingFailBackend) Ping(ctx context.Context) error {
	p.calls++
	if p.calls < p.okAfter {
		return errors.New("ping fail")
	}
	return p.memBackend.Ping(ctx)
}

func TestBuildBackend_cosURL(t *testing.T) {
	cfg := COSConfig{SecretID: "a", SecretKey: "b", Bucket: "mybucket", Region: "ap-guangzhou"}
	u, err := cosBucketURL(cfg)
	if err != nil || u.Host == "" {
		t.Fatal(err, u)
	}
}
