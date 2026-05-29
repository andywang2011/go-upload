package upload

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStorage_uploadDownload_list(t *testing.T) {
	for _, p := range []string{ProviderAliyun, ProviderTencent, ProviderWechat, ProviderAWS, ProviderAzure} {
		t.Run(p, func(t *testing.T) {
			s := mustTestStorage(t, p)
			defer s.Close()
			ctx := context.Background()
			body := []byte("hello-upload")
			if err := s.Upload(ctx, "a/b.txt", bytes.NewReader(body), WithContentLength(int64(len(body)))); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := s.Download(ctx, "a/b.txt", &buf); err != nil {
				t.Fatal(err)
			}
			if buf.String() != string(body) {
				t.Fatalf("got %q", buf.String())
			}
			objs, page, err := s.List(ctx, "a/")
			if err != nil || page == nil {
				t.Fatal(err, page)
			}
			if len(objs) != 1 || objs[0].Key != "a/b.txt" {
				t.Fatalf("objs=%+v", objs)
			}
			if err := s.Delete(ctx, "a/b.txt"); err != nil {
				t.Fatal(err)
			}
			_ = s.PublicURL("a/b.txt")
			if err := s.UploadLarge(ctx, "big", bytes.NewReader([]byte("large")), 5); err != nil {
				t.Fatal(err)
			}
			var big bytes.Buffer
			_ = s.DownloadLarge(ctx, "big", &big, 5)
		})
	}
}

func TestSetMemObject_errors(t *testing.T) {
	if err := SetMemObject(struct{ Storage }{nil}, "k", nil); err == nil {
		t.Fatal()
	}
}

func TestStorage_uploadLarge_threshold(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	defer s.Close()
	cfg := validTestConfig(ProviderAliyun)
	cfg.Multipart.Threshold = 4
	st := newStorage(cfg, newMemBackend("https://cdn"))
	ctx := context.Background()
	data := []byte("12345678")
	if err := st.Upload(ctx, "big.bin", bytes.NewReader(data), WithContentLength(int64(len(data)))); err != nil {
		t.Fatal(err)
	}
}

func TestStorage_notInitialized(t *testing.T) {
	var s *storage
	if err := s.Upload(context.Background(), "k", nil); err != ErrNotInitialized {
		t.Fatal(err)
	}
	if s.PublicURL("k") != "" {
		t.Fatal()
	}
	if _, _, err := s.List(context.Background(), ""); err != ErrNotInitialized {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewStorage_invalidConfig(t *testing.T) {
	_, err := NewStorage(Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewStorage_pingFails(t *testing.T) {
	old := buildBackend
	buildBackend = func(cfg Config) (objectBackend, error) {
		return &pingFailBackend{memBackend: newMemBackend("https://cdn"), okAfter: 100}, nil
	}
	defer func() { buildBackend = old }()
	cfg := validTestConfig(ProviderAliyun)
	cfg.Retry = RetryConfig{MaxAttempts: 2, InitialDelay: 1, MaxDelay: 1, Multiplier: 1}
	_, err := NewStorage(cfg)
	if err == nil {
		t.Fatal("expected ping error")
	}
}

func TestNewStorage_withMemBackendHook(t *testing.T) {
	old := buildBackend
	buildBackend = func(cfg Config) (objectBackend, error) {
		return newMemBackend("https://mem"), nil
	}
	defer func() { buildBackend = old }()
	s, err := NewStorage(validTestConfig(ProviderAliyun))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !strings.HasPrefix(s.PublicURL("x"), "https://mem") {
		t.Fatal(s.PublicURL("x"))
	}
}

func TestStorage_PublicURL_and_AsUploader(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	u := AsUploader(s)
	if u.CdnUrl("p") != s.PublicURL("p") {
		t.Fatal()
	}
	if u.BaseURL() != "https://oss.cdn" {
		t.Fatalf("base=%q", u.BaseURL())
	}
}

func TestStorage_downloadUsesMultipartThreshold(t *testing.T) {
	cfg := validTestConfig(ProviderAliyun)
	cfg.Multipart.Threshold = 5
	s, err := testStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	payload := bytes.Repeat([]byte("x"), 20)
	if err := s.Upload(ctx, "f", bytes.NewReader(payload), WithContentLength(int64(len(payload)))); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.Download(ctx, "f", &buf); err != nil {
		t.Fatal(err)
	}
}

func TestStorage_downloadLarge_head(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	defer s.Close()
	ctx := context.Background()
	data := []byte("1234567890")
	_ = s.Upload(ctx, "d.bin", bytes.NewReader(data), WithContentLength(int64(len(data))))
	var buf bytes.Buffer
	if err := s.DownloadLarge(ctx, "d.bin", &buf, 0); err != nil {
		t.Fatal(err)
	}
}

func TestStorage_uploadLarge_downloadLarge(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	defer s.Close()
	ctx := context.Background()
	data := []byte("large-payload-1234567890")
	if err := s.UploadLarge(ctx, "big.obj", bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.DownloadLarge(ctx, "big.obj", &buf, int64(len(data))); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != len(data) {
		t.Fatal(buf.Len())
	}
}

func TestAsUploader_upload(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	u := AsUploader(s)
	if err := u.Upload(context.Background(), "u.txt", bytes.NewReader([]byte("u"))); err != nil {
		t.Fatal(err)
	}
}

func TestSetMemObject(t *testing.T) {
	s := mustTestStorage(t, ProviderAliyun)
	if err := SetMemObject(s, "seed", []byte("x")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.Download(context.Background(), "seed", &buf); err != nil {
		t.Fatal(err)
	}
}
