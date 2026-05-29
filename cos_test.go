package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	cos "github.com/tencentyun/cos-go-sdk-v5"
)

type mockCOS struct {
	objects map[string][]byte
}

func (m *mockCOS) BucketGet(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
	res := &cos.BucketGetResult{}
	for k, v := range m.objects {
		if opt != nil && opt.Prefix != "" && len(k) < len(opt.Prefix) {
			continue
		}
		res.Contents = append(res.Contents, cos.Object{Key: k, Size: int64(len(v))})
	}
	return res, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
}

func (m *mockCOS) ObjectPut(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	data, _ := io.ReadAll(r)
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[name] = data
	return &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
}

func (m *mockCOS) ObjectGet(ctx context.Context, name string, opt *cos.ObjectGetOptions) (*cos.Response, error) {
	data, ok := m.objects[name]
	if !ok {
		return nil, errors.New("404 NoSuchKey")
	}
	return &cos.Response{Response: &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data))}}, nil
}

func (m *mockCOS) ObjectHead(ctx context.Context, name string, opt *cos.ObjectHeadOptions) (*cos.Response, error) {
	data, ok := m.objects[name]
	if !ok {
		return nil, errors.New("404")
	}
	h := make(http.Header)
	h.Set("Content-Length", strconv.Itoa(len(data)))
	return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
}

func (m *mockCOS) ObjectDelete(ctx context.Context, name string) (*cos.Response, error) {
	delete(m.objects, name)
	return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
}

func (m *mockCOS) ObjectInitiateMultipartUpload(ctx context.Context, name string, opt *cos.InitiateMultipartUploadOptions) (*cos.InitiateMultipartUploadResult, *cos.Response, error) {
	return &cos.InitiateMultipartUploadResult{UploadID: "up"}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
}

func (m *mockCOS) ObjectUploadPart(ctx context.Context, name, uploadID string, partNumber int, r io.Reader, opt *cos.ObjectUploadPartOptions) (*cos.Response, error) {
	_, _ = io.ReadAll(r)
	h := make(http.Header)
	h.Set("ETag", `"etag"`)
	return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
}

func (m *mockCOS) ObjectCompleteMultipartUpload(ctx context.Context, name, uploadID string, opt *cos.CompleteMultipartUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
	return &cos.CompleteMultipartUploadResult{}, &cos.Response{Response: &http.Response{StatusCode: 200}}, nil
}

func (m *mockCOS) ObjectAbortMultipartUpload(ctx context.Context, name, uploadID string) (*cos.Response, error) {
	return &cos.Response{Response: &http.Response{StatusCode: 204}}, nil
}

func (m *mockCOS) BucketBaseURL() string { return "https://cos.example" }

func TestCOSBackend_full(t *testing.T) {
	mock := &mockCOS{objects: map[string][]byte{}}
	b := &cosBackend{client: mock, cfg: COSConfig{CDNHost: "https://cdn"}, mp: MultipartConfig{PartSize: 4}}
	ctx := context.Background()
	if err := b.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Put(ctx, "k", bytes.NewReader([]byte("data")), 4, ""); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := b.Get(ctx, "k", &buf); err != nil {
		t.Fatal(err)
	}
	if err := b.UploadMultipart(ctx, "big", bytes.NewReader([]byte("12345678")), 8, b.mp); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.List(ctx, "", listOptions{}); err != nil {
		t.Fatal(err)
	}
	var buf2 bytes.Buffer
	if err := b.GetRange(ctx, "k", &buf2, 0, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Head(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if b.PublicURL("k") != "https://cdn/k" {
		t.Fatal(b.PublicURL("k"))
	}
	mock2 := &mockCOS{objects: map[string][]byte{"big": []byte("12345678")}}
	b2 := &cosBackend{client: mock2, cfg: COSConfig{}, mp: MultipartConfig{PartSize: 4, DownloadPartSize: 4}}
	var dl bytes.Buffer
	if err := b2.DownloadMultipart(ctx, "big", &dl, 8, b2.mp); err != nil {
		t.Fatal(err)
	}
	_ = b.Delete(ctx, "k")
	_ = b.Close()
}

func TestCOSBackend_publicURL_base(t *testing.T) {
	mock := &mockCOS{objects: map[string][]byte{}}
	b := &cosBackend{client: mock, cfg: COSConfig{}, mp: ConfigDefault().Multipart}
	if !strings.Contains(b.PublicURL("x"), "cos.example") {
		t.Fatal(b.PublicURL("x"))
	}
}

func TestMapCOSErr(t *testing.T) {
	if mapCOSErr(nil) != nil {
		t.Fatal()
	}
	if mapCOSErr(errNotFoundLike{msg: "404 Not Found"}) != ErrNotFound {
		t.Fatal()
	}
	er := &cos.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
		Message:  "Not Found",
	}
	if mapCOSErr(er) != ErrNotFound {
		t.Fatal()
	}
}

func TestCOSBackend_headNotFound(t *testing.T) {
	b := &cosBackend{client: &mockCOS{objects: map[string][]byte{}}, cfg: COSConfig{}, mp: ConfigDefault().Multipart}
	if _, err := b.Head(context.Background(), "nope"); err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestNewCOSBackend_hook(t *testing.T) {
	old := newCOSClient
	newCOSClient = func(cfg COSConfig) (cosAPI, error) {
		return &mockCOS{objects: map[string][]byte{}}, nil
	}
	defer func() { newCOSClient = old }()
	backend, err := newCOSBackend(COSConfig{SecretID: "a", SecretKey: "b", Bucket: "bk"}, ConfigDefault().Multipart)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
