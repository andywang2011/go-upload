package upload

import (
	"io"
	"net/http"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

var (
	newOSSBucket   = defaultNewOSSBucket
	newCOSClient   = defaultNewCOSClient
	newAWSClient   = defaultNewAWSClient
	newAzureClient = defaultNewAzureClient
)

func defaultNewOSSBucket(cfg AliyunConfig) (ossBucketAPI, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, oss.AuthVersion(oss.AuthV4))
	if err != nil {
		return nil, err
	}
	return client.Bucket(cfg.Bucket)
}

func defaultNewCOSClient(cfg COSConfig) (cosAPI, error) {
	u, err := cosBucketURL(cfg)
	if err != nil {
		return nil, err
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return &cosClientAdapter{c}, nil
}

// ossBucketAPI is the OSS operations used by the driver (mockable in tests).
type ossBucketAPI interface {
	PutObject(key string, reader io.Reader, options ...oss.Option) error
	GetObject(key string, options ...oss.Option) (io.ReadCloser, error)
	DeleteObject(key string, options ...oss.Option) error
	ListObjectsV2(options ...oss.Option) (oss.ListObjectsResultV2, error)
	GetObjectMeta(key string, options ...oss.Option) (http.Header, error)
	InitiateMultipartUpload(key string, options ...oss.Option) (oss.InitiateMultipartUploadResult, error)
	UploadPart(imur oss.InitiateMultipartUploadResult, reader io.Reader, size int64, partNumber int, options ...oss.Option) (oss.UploadPart, error)
	CompleteMultipartUpload(imur oss.InitiateMultipartUploadResult, parts []oss.UploadPart, options ...oss.Option) (oss.CompleteMultipartUploadResult, error)
	AbortMultipartUpload(imur oss.InitiateMultipartUploadResult, options ...oss.Option) error
}
