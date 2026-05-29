package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type mockAWS struct {
	bucket  string
	objects map[string][]byte
}

func (m *mockAWS) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}

func (m *mockAWS) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, _ := io.ReadAll(params.Body)
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[aws.ToString(params.Key)] = data
	return &s3.PutObjectOutput{}, nil
}

func (m *mockAWS) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := m.objects[aws.ToString(params.Key)]
	if !ok {
		return nil, errors.New("NoSuchKey")
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (m *mockAWS) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var contents []types.Object
	for k, v := range m.objects {
		kk, sz := k, int64(len(v))
		contents = append(contents, types.Object{Key: &kk, Size: &sz})
	}
	trunc := false
	return &s3.ListObjectsV2Output{Contents: contents, IsTruncated: &trunc}, nil
}

func (m *mockAWS) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	delete(m.objects, aws.ToString(params.Key))
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockAWS) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	data, ok := m.objects[aws.ToString(params.Key)]
	if !ok {
		return nil, errors.New("NotFound")
	}
	sz := int64(len(data))
	return &s3.HeadObjectOutput{ContentLength: &sz}, nil
}

func (m *mockAWS) CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	id := "upload-id"
	return &s3.CreateMultipartUploadOutput{UploadId: &id}, nil
}

func (m *mockAWS) UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	data, _ := io.ReadAll(params.Body)
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[aws.ToString(params.Key)] = append(m.objects[aws.ToString(params.Key)], data...)
	etag := "etag"
	return &s3.UploadPartOutput{ETag: &etag}, nil
}

func (m *mockAWS) CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (m *mockAWS) AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}

func TestAWSBackend_full(t *testing.T) {
	mock := &mockAWS{objects: map[string][]byte{}}
	b := &awsBackend{
		client: mock,
		cfg:    AWSConfig{Bucket: "bk", CDNHost: "https://aws.cdn"},
		mp:     MultipartConfig{PartSize: 4, Threshold: 8},
	}
	ctx := context.Background()
	if err := b.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Put(ctx, "k", bytes.NewReader([]byte("abcd")), 4, "text/plain"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := b.Get(ctx, "k", &buf); err != nil {
		t.Fatal(err)
	}
	if err := b.GetRange(ctx, "k", &buf, 0, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Head(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.List(ctx, "", listOptions{maxKeys: 10}); err != nil {
		t.Fatal(err)
	}
	if err := b.UploadMultipart(ctx, "big", bytes.NewReader([]byte("12345678")), 8, b.mp); err != nil {
		t.Fatal(err)
	}
	var dl bytes.Buffer
	if err := b.DownloadMultipart(ctx, "big", &dl, 8, b.mp); err != nil {
		t.Fatal(err)
	}
	if err := b.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if b.PublicURL("x") != "https://aws.cdn/x" {
		t.Fatal(b.PublicURL("x"))
	}
	_ = b.Close()
}

func TestAWSBackend_getNotFound(t *testing.T) {
	b := &awsBackend{client: &mockAWS{objects: map[string][]byte{}}, cfg: AWSConfig{Bucket: "bk"}, mp: ConfigDefault().Multipart}
	var buf bytes.Buffer
	if err := b.Get(context.Background(), "missing", &buf); err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestAWSBackend_mapNotFound(t *testing.T) {
	if mapAWSErr(nil) != nil {
		t.Fatal()
	}
	err := mapAWSErr(errNotFoundLike{msg: "NoSuchKey"})
	if err != ErrNotFound {
		t.Fatal(err)
	}
}

func TestNewAWSClientFromConfig_customEndpoint(t *testing.T) {
	_, err := newAWSClientFromConfig(AWSConfig{
		Region: "us-east-1", AccessKeyID: "a", SecretAccessKey: "b", Bucket: "mybucket",
		Endpoint: "http://127.0.0.1:9000",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewAWSBackend_hook(t *testing.T) {
	old := newAWSClient
	newAWSClient = func(cfg AWSConfig) (awsAPI, error) {
		return &mockAWS{objects: map[string][]byte{}}, nil
	}
	defer func() { newAWSClient = old }()
	backend, err := newAWSBackend(AWSConfig{Region: "us-east-1", AccessKeyID: "a", SecretAccessKey: "b", Bucket: "bk"}, ConfigDefault().Multipart)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
