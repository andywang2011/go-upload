package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// memBackend is an in-memory objectBackend for unit tests.
type memBackend struct {
	mu      sync.RWMutex
	objects map[string][]byte
	cdnHost string
	closed  bool
}

func newMemBackend(cdnHost string) *memBackend {
	return &memBackend{
		objects: make(map[string][]byte),
		cdnHost: cdnHost,
	}
}

func (m *memBackend) Ping(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *memBackend) Put(ctx context.Context, key string, r io.Reader, size int64, _ string) error {
	_ = ctx
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memBackend) Get(ctx context.Context, key string, w io.Writer) error {
	_ = ctx
	m.mu.RLock()
	data, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	_, err := io.Copy(w, bytes.NewReader(data))
	return err
}

func (m *memBackend) GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error {
	_ = ctx
	m.mu.RLock()
	data, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	if offset >= int64(len(data)) {
		return io.EOF
	}
	end := offset + length
	if end > int64(len(data)) {
		end = int64(len(data))
	}
	_, err := w.Write(data[offset:end])
	return err
}

func (m *memBackend) List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []ObjectInfo
	for k, v := range m.objects {
		if prefix != "" && !hasPrefix(k, prefix) {
			continue
		}
		out = append(out, ObjectInfo{Key: k, Size: int64(len(v))})
	}
	if opts.maxKeys > 0 && len(out) > opts.maxKeys {
		out = out[:opts.maxKeys]
		return out, &ListPage{IsTruncated: true}, nil
	}
	return out, &ListPage{}, nil
}

func (m *memBackend) Delete(ctx context.Context, key string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[key]; !ok {
		return ErrNotFound
	}
	delete(m.objects, key)
	return nil
}

func (m *memBackend) Head(ctx context.Context, key string) (int64, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.objects[key]
	if !ok {
		return 0, ErrNotFound
	}
	return int64(len(data)), nil
}

func (m *memBackend) UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	return m.Put(ctx, key, r, size, "")
}

func (m *memBackend) DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
	if size <= 0 {
		size, _ = m.Head(ctx, key)
	}
	partSize := mp.DownloadPartSize
	if partSize <= 0 {
		partSize = mp.PartSize
	}
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	for off := int64(0); off < size; off += partSize {
		n := partSize
		if off+n > size {
			n = size - off
		}
		if err := m.GetRange(ctx, key, w, off, n); err != nil {
			return err
		}
	}
	return nil
}

func (m *memBackend) PublicURL(key string) string {
	return PublicURLFromHost(m.cdnHost, key)
}

func (m *memBackend) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// testStorage builds Storage backed by memBackend (tests only).
func testStorage(cfg Config) (Storage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	host := "https://cdn.test"
	switch cfg.Provider.Name {
	case ProviderAliyun:
		host = cfg.Aliyun.CDNHost
	case ProviderTencent:
		host = cfg.Tencent.CDNHost
	case ProviderWechat:
		host = cfg.Wechat.CDNHost
	case ProviderAWS:
		host = cfg.AWS.CDNHost
	case ProviderAzure:
		host = cfg.Azure.CDNHost
	}
	if host == "" {
		host = "https://cdn.test"
	}
	return newStorage(cfg, newMemBackend(host)), nil
}

func validTestConfig(provider string) Config {
	cfg := ConfigDefault()
	cfg.Provider.Name = provider
	cfg.Aliyun = AliyunConfig{
		Endpoint: "oss.test", AccessKeyID: "a", AccessKeySecret: "b", Bucket: "mybucket", CDNHost: "https://oss.cdn",
	}
	cfg.Tencent = COSConfig{Region: "ap-shanghai", SecretID: "a", SecretKey: "b", Bucket: "mybucket"}
	cfg.Wechat = COSConfig{Region: "ap-shanghai", SecretID: "a", SecretKey: "b", Bucket: "mybucket"}
	cfg.AWS = AWSConfig{Region: "us-east-1", AccessKeyID: "a", SecretAccessKey: "b", Bucket: "mybucket"}
	cfg.Azure = AzureConfig{AccountName: "acct", AccountKey: "key", Container: "ctr"}
	cfg.Retry.MaxAttempts = 1
	return cfg
}

func mustTestStorage(t interface {
	Helper()
	Fatal(...any)
}, provider string) Storage {
	t.Helper()
	s, err := testStorage(validTestConfig(provider))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// SetMemObject seeds mem backend when buildBackend is overridden in tests.
func SetMemObject(s Storage, key string, data []byte) error {
	st, ok := s.(*storage)
	if !ok {
		return fmt.Errorf("not a storage wrapper")
	}
	mb, ok := st.backend.(*memBackend)
	if !ok {
		return fmt.Errorf("backend is not memBackend")
	}
	mb.mu.Lock()
	mb.objects[key] = append([]byte(nil), data...)
	mb.mu.Unlock()
	return nil
}
