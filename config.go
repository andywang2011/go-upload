package upload

import (
	"fmt"
	"time"
)

const (
	ProviderAliyun  = "aliyun"
	ProviderTencent = "tencent"
	ProviderWechat  = "wechat"
	ProviderAWS     = "aws"
	ProviderAzure   = "azure"
)

const (
	defaultPartSize      = 8 * 1024 * 1024
	defaultConcurrency   = 4
	defaultThreshold     = 100 * 1024 * 1024
	defaultTencentRegion = "ap-shanghai"
)

// MultipartConfig controls chunked transfers.
type MultipartConfig struct {
	// PartSize is the upload part size in bytes.
	PartSize int64
	// Concurrency is the number of parallel part uploads.
	Concurrency int
	// Threshold is the file size above which multipart upload is used.
	Threshold int64
	// DownloadPartSize is the chunk size for ranged downloads.
	DownloadPartSize int64
}

// RetryConfig controls startup connectivity retry.
type RetryConfig struct {
	// MaxAttempts is the total number of connection attempts.
	MaxAttempts int
	// InitialDelay is the base delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay caps exponential backoff delay.
	MaxDelay time.Duration
	// Multiplier scales delay after each failed attempt.
	Multiplier float64
}

// Config holds provider credentials and tuning.
type Config struct {
	// Provider selects the active storage backend.
	Provider ProviderSettings
	// Aliyun holds Aliyun OSS credentials when Provider is aliyun.
	Aliyun AliyunConfig
	// Tencent holds Tencent COS credentials when Provider is tencent.
	Tencent COSConfig
	// Wechat holds WeChat COS credentials when Provider is wechat.
	Wechat COSConfig
	// AWS holds Amazon S3 credentials when Provider is aws.
	AWS AWSConfig
	// Azure holds Azure Blob credentials when Provider is azure.
	Azure AzureConfig
	// Multipart tunes chunked upload/download behavior.
	Multipart MultipartConfig
	// Retry controls startup connectivity retry.
	Retry RetryConfig
}

// ProviderSettings selects active backend.
type ProviderSettings struct {
	// Name is one of ProviderAliyun, ProviderTencent, ProviderWechat, ProviderAWS, ProviderAzure.
	Name string
}

// AliyunConfig is Aliyun OSS settings.
type AliyunConfig struct {
	// Endpoint is the OSS endpoint URL.
	Endpoint string
	// AccessKeyID is the Aliyun access key ID.
	AccessKeyID string
	// AccessKeySecret is the Aliyun access key secret.
	AccessKeySecret string
	// Bucket is the target OSS bucket name.
	Bucket string
	// CDNHost is the optional CDN base URL for public object URLs.
	CDNHost string
}

// COSConfig is Tencent/WeChat COS settings.
type COSConfig struct {
	// Region is the COS region (e.g. ap-shanghai).
	Region string
	// SecretID is the Tencent Cloud secret ID.
	SecretID string
	// SecretKey is the Tencent Cloud secret key.
	SecretKey string
	// Bucket is the bucket name without app ID suffix.
	Bucket string
	// AppID is the Tencent Cloud app ID appended to the bucket.
	AppID string
	// CDNHost is the optional CDN base URL for public object URLs.
	CDNHost string
	// BaseURL is an optional custom API base URL.
	BaseURL string
}

// AWSConfig is Amazon S3 settings.
type AWSConfig struct {
	// Region is the AWS region (e.g. us-east-1).
	Region string
	// AccessKeyID is the AWS access key ID.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
	// SessionToken is the optional STS session token.
	SessionToken string
	// Bucket is the S3 bucket name.
	Bucket string
	// Endpoint is an optional custom S3-compatible endpoint.
	Endpoint string
	// CDNHost is the optional CDN base URL for public object URLs.
	CDNHost string
}

// AzureConfig is Azure Blob settings.
type AzureConfig struct {
	// AccountName is the storage account name.
	AccountName string
	// AccountKey is the storage account access key.
	AccountKey string
	// Container is the blob container name.
	Container string
	// Endpoint is an optional custom blob endpoint.
	Endpoint string
	// CDNHost is the optional CDN base URL for public object URLs.
	CDNHost string
}

// ConfigDefault returns tuning defaults (provider block must still be set).
func ConfigDefault() Config {
	return Config{
		Multipart: MultipartConfig{
			PartSize:         defaultPartSize,
			Concurrency:      defaultConcurrency,
			Threshold:        defaultThreshold,
			DownloadPartSize: defaultPartSize,
		},
		Retry: DefaultRetryConfig(),
	}
}

// DefaultRetryConfig returns startup retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// Validate applies defaults and checks provider-specific required fields.
func (c *Config) Validate() error {
	if c.Provider.Name == "" {
		return fmt.Errorf("%w: provider name is required", ErrInvalidConfig)
	}
	if c.Multipart.PartSize <= 0 {
		c.Multipart.PartSize = defaultPartSize
	}
	if c.Multipart.Concurrency <= 0 {
		c.Multipart.Concurrency = defaultConcurrency
	}
	if c.Multipart.Threshold <= 0 {
		c.Multipart.Threshold = defaultThreshold
	}
	if c.Multipart.DownloadPartSize <= 0 {
		c.Multipart.DownloadPartSize = c.Multipart.PartSize
	}
	if c.Retry.MaxAttempts <= 0 {
		c.Retry = DefaultRetryConfig()
	} else {
		normalizeRetry(&c.Retry)
	}
	switch c.Provider.Name {
	case ProviderAliyun:
		if !c.Aliyun.valid() {
			return fmt.Errorf("%w: aliyun endpoint, keys, and bucket required", ErrInvalidConfig)
		}
	case ProviderTencent:
		if !c.Tencent.valid() {
			return fmt.Errorf("%w: tencent cos region, keys, and bucket required", ErrInvalidConfig)
		}
	case ProviderWechat:
		if !c.Wechat.valid() {
			return fmt.Errorf("%w: wechat cos region, keys, and bucket required", ErrInvalidConfig)
		}
	case ProviderAWS:
		if !c.AWS.valid() {
			return fmt.Errorf("%w: aws region, keys, and bucket required", ErrInvalidConfig)
		}
	case ProviderAzure:
		if !c.Azure.valid() {
			return fmt.Errorf("%w: azure account, key, and container required", ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: unknown provider %q", ErrInvalidConfig, c.Provider.Name)
	}
	return nil
}

func (a AliyunConfig) valid() bool {
	return a.Endpoint != "" && a.AccessKeyID != "" && a.AccessKeySecret != "" && a.Bucket != ""
}

func (c COSConfig) valid() bool {
	return c.SecretID != "" && c.SecretKey != "" && c.Bucket != ""
}

func (a AWSConfig) valid() bool {
	return a.Region != "" && a.AccessKeyID != "" && a.SecretAccessKey != "" && a.Bucket != ""
}

func (a AzureConfig) valid() bool {
	return a.AccountName != "" && a.AccountKey != "" && a.Container != ""
}

func (c COSConfig) regionOrDefault() string {
	if c.Region != "" {
		return c.Region
	}
	return defaultTencentRegion
}
