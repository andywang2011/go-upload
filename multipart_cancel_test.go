package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type mockOSSPartFail struct{ mockOSSBucket }

func (m *mockOSSPartFail) UploadPart(imur oss.InitiateMultipartUploadResult, reader io.Reader, size int64, partNumber int, _ ...oss.Option) (oss.UploadPart, error) {
	return oss.UploadPart{}, errors.New("part failed")
}

func TestOSSBackend_multipartPartFail(t *testing.T) {
	b := &ossBackend{
		bucket: &mockOSSPartFail{mockOSSBucket: mockOSSBucket{objects: map[string][]byte{}}},
		cfg:    AliyunConfig{},
		mp:     MultipartConfig{PartSize: 4},
	}
	err := b.UploadMultipart(context.Background(), "k", bytes.NewReader([]byte("12345678")), 8, b.mp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCOSBackend_uploadMultipartCanceled(t *testing.T) {
	b := &cosBackend{client: &mockCOS{objects: map[string][]byte{}}, cfg: COSConfig{}, mp: MultipartConfig{PartSize: 4}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.UploadMultipart(ctx, "k", bytes.NewReader([]byte("12345678")), 8, b.mp); err == nil {
		t.Fatal()
	}
}

func TestAzureBackend_uploadMultipartCanceled(t *testing.T) {
	b := &azureBackend{container: "ctr", cfg: AzureConfig{}, mp: MultipartConfig{PartSize: 4}}
	restore := saveAzureHooks()
	defer restore()
	newAzureBlockBlob = func(ab *azureBackend, key string) blockBlobAPI { return &mockAzureBlob{data: map[string][]byte{}} }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := azureUploadMultipart(b, ctx, "obj", bytes.NewReader([]byte("12345678")), 8, b.mp); err == nil {
		t.Fatal()
	}
}

func TestAWSBackend_uploadMultipartCanceled(t *testing.T) {
	b := &awsBackend{client: &mockAWS{objects: map[string][]byte{}}, cfg: AWSConfig{Bucket: "bk"}, mp: MultipartConfig{PartSize: 4}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.UploadMultipart(ctx, "k", bytes.NewReader([]byte("12345678")), 8, b.mp); err == nil {
		t.Fatal()
	}
}
