package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type mockOSSBucket struct {
	objects map[string][]byte
	putErr  error
	listErr error
}

func (m *mockOSSBucket) PutObject(key string, reader io.Reader, _ ...oss.Option) error {
	if m.putErr != nil {
		return m.putErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	if m.objects == nil {
		m.objects = make(map[string][]byte)
	}
	m.objects[key] = data
	return nil
}

func (m *mockOSSBucket) GetObject(key string, _ ...oss.Option) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockOSSBucket) DeleteObject(key string, _ ...oss.Option) error {
	delete(m.objects, key)
	return nil
}

func (m *mockOSSBucket) ListObjectsV2(_ ...oss.Option) (oss.ListObjectsResultV2, error) {
	if m.listErr != nil {
		return oss.ListObjectsResultV2{}, m.listErr
	}
	var res oss.ListObjectsResultV2
	for k := range m.objects {
		res.Objects = append(res.Objects, oss.ObjectProperties{Key: k, Size: int64(len(m.objects[k]))})
	}
	return res, nil
}

func (m *mockOSSBucket) GetObjectMeta(key string, _ ...oss.Option) (http.Header, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	h := make(http.Header)
	h.Set("Content-Length", strconv.Itoa(len(data)))
	return h, nil
}

func (m *mockOSSBucket) InitiateMultipartUpload(key string, _ ...oss.Option) (oss.InitiateMultipartUploadResult, error) {
	return oss.InitiateMultipartUploadResult{Bucket: "b", Key: key, UploadID: "up"}, nil
}

func (m *mockOSSBucket) UploadPart(imur oss.InitiateMultipartUploadResult, reader io.Reader, size int64, partNumber int, _ ...oss.Option) (oss.UploadPart, error) {
	if err := m.PutObject(imur.Key, reader); err != nil {
		return oss.UploadPart{}, err
	}
	return oss.UploadPart{PartNumber: partNumber, ETag: "etag"}, nil
}

func (m *mockOSSBucket) CompleteMultipartUpload(imur oss.InitiateMultipartUploadResult, _ []oss.UploadPart, _ ...oss.Option) (oss.CompleteMultipartUploadResult, error) {
	return oss.CompleteMultipartUploadResult{Key: imur.Key}, nil
}

func (m *mockOSSBucket) AbortMultipartUpload(oss.InitiateMultipartUploadResult, ...oss.Option) error {
	return nil
}

func TestOSSBackend_putError(t *testing.T) {
	mock := &mockOSSBucket{putErr: errors.New("put failed")}
	b := &ossBackend{bucket: mock, cfg: AliyunConfig{}, mp: ConfigDefault().Multipart}
	if err := b.Put(context.Background(), "k", bytes.NewReader(nil), 0, ""); err == nil {
		t.Fatal()
	}
}

func TestOSSBackend_pingListError(t *testing.T) {
	mock := &mockOSSBucket{listErr: errors.New("list failed")}
	b := &ossBackend{bucket: mock, cfg: AliyunConfig{}, mp: ConfigDefault().Multipart}
	if err := b.Ping(context.Background()); err == nil {
		t.Fatal()
	}
}

func TestOSSBackend_putGetList(t *testing.T) {
	mock := &mockOSSBucket{objects: make(map[string][]byte)}
	b := &ossBackend{
		bucket: mock,
		cfg:    AliyunConfig{CDNHost: "https://cdn"},
		mp:     ConfigDefault().Multipart,
	}
	ctx := context.Background()
	if err := b.Put(ctx, "k", bytes.NewReader([]byte("data")), 4, "text/plain"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := b.Get(ctx, "k", &buf); err != nil {
		t.Fatal(err)
	}
	objs, page, err := b.List(ctx, "k", listOptions{maxKeys: 10, delimiter: "/", continuationToken: "t"})
	if err != nil || page == nil || len(objs) != 1 {
		t.Fatal(err, objs, page)
	}
	if b.PublicURL("k") != "https://cdn/k" {
		t.Fatal(b.PublicURL("k"))
	}
}

func TestMapOSSErr(t *testing.T) {
	if mapOSSErr(nil) != nil {
		t.Fatal()
	}
	if mapOSSErr(errNotFoundLike{msg: "NoSuchKey status 404"}) != ErrNotFound {
		t.Fatal()
	}
}

func TestOSSBackend_mapNotFound(t *testing.T) {
	b := &ossBackend{bucket: &mockOSSBucket{}, cfg: AliyunConfig{}, mp: ConfigDefault().Multipart}
	var buf bytes.Buffer
	if err := b.Get(context.Background(), "missing", &buf); err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestOSSBackend_multipartUnknownSize(t *testing.T) {
	mock := &mockOSSBucket{objects: make(map[string][]byte)}
	b := &ossBackend{bucket: mock, cfg: AliyunConfig{}, mp: MultipartConfig{PartSize: 2}}
	if err := b.UploadMultipart(context.Background(), "m", bytes.NewReader([]byte("abc")), 0, b.mp); err != nil {
		t.Fatal(err)
	}
}

func TestOSSBackend_multipartAndRange(t *testing.T) {
	mock := &mockOSSBucket{objects: make(map[string][]byte)}
	b := &ossBackend{
		bucket: mock,
		cfg:    AliyunConfig{},
		mp:     MultipartConfig{PartSize: 4},
	}
	ctx := context.Background()
	data := []byte("12345678")
	if err := b.UploadMultipart(ctx, "m", bytes.NewReader(data), int64(len(data)), b.mp); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := b.DownloadMultipart(ctx, "m", &buf, int64(len(data)), b.mp); err != nil {
		t.Fatal(err)
	}
	if err := b.GetRange(ctx, "m", &bytes.Buffer{}, 0, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Head(ctx, "m"); err != nil {
		t.Fatal(err)
	}
	_ = b.Delete(ctx, "m")
}

func TestNewOSSBackend_hook(t *testing.T) {
	old := newOSSBucket
	newOSSBucket = func(AliyunConfig) (ossBucketAPI, error) {
		return &mockOSSBucket{objects: make(map[string][]byte)}, nil
	}
	defer func() { newOSSBucket = old }()
	backend, err := newOSSBackend(AliyunConfig{Endpoint: "e", AccessKeyID: "a", AccessKeySecret: "b", Bucket: "bk"}, ConfigDefault().Multipart)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOSSBackend_close(t *testing.T) {
	b := &ossBackend{bucket: &mockOSSBucket{}, cfg: AliyunConfig{}, mp: ConfigDefault().Multipart}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFactory_buildBackend_aliyun(t *testing.T) {
	old := newOSSBucket
	newOSSBucket = func(AliyunConfig) (ossBucketAPI, error) {
		return &mockOSSBucket{objects: make(map[string][]byte)}, nil
	}
	defer func() { newOSSBucket = old }()
	cfg := validTestConfig(ProviderAliyun)
	backend, err := buildBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := backend.(*ossBackend); !ok {
		t.Fatalf("%T", backend)
	}
}
