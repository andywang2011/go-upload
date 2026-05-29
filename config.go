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
	defaultPartSize     = 8 * 1024 * 1024
	defaultConcurrency  = 4
	defaultThreshold    = 100 * 1024 * 1024
	defaultTencentRegion = "ap-shanghai"
)

// MultipartConfig controls chunked transfers.
type MultipartConfig struct {
	PartSize         int64
	Concurrency      int
	Threshold        int64
	DownloadPartSize int64
}

// RetryConfig controls startup connectivity retry.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// Config holds provider credentials and tuning.
type Config struct {
	Provider ProviderSettings
	Aliyun   AliyunConfig
	Tencent  COSConfig
	Wechat   COSConfig
	AWS      AWSConfig
	Azure    AzureConfig
	Multipart MultipartConfig
	Retry    RetryConfig
}

// ProviderSettings selects active backend.
type ProviderSettings struct {
	Name string
}

// AliyunConfig is Aliyun OSS settings.
type AliyunConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	CDNHost         string
}

// COSConfig is Tencent/WeChat COS settings.
type COSConfig struct {
	Region          string
	SecretID        string
	SecretKey       string
	Bucket          string
	AppID           string
	CDNHost         string
	BaseURL         string
}

// AWSConfig is Amazon S3 settings.
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Bucket          string
	Endpoint        string
	CDNHost         string
}

// AzureConfig is Azure Blob settings.
type AzureConfig struct {
	AccountName   string
	AccountKey    string
	Container     string
	Endpoint      string
	CDNHost       string
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
