package upload

import (
	"testing"
	"time"
)

func TestConfig_Validate_allProviders(t *testing.T) {
	providers := []struct {
		name string
		cfg  func(*Config)
	}{
		{ProviderAliyun, func(c *Config) { c.Aliyun.Endpoint = "e"; c.Aliyun.AccessKeyID = "a"; c.Aliyun.AccessKeySecret = "b"; c.Aliyun.Bucket = "bk" }},
		{ProviderTencent, func(c *Config) { c.Tencent.SecretID = "a"; c.Tencent.SecretKey = "b"; c.Tencent.Bucket = "bk" }},
		{ProviderWechat, func(c *Config) { c.Wechat.SecretID = "a"; c.Wechat.SecretKey = "b"; c.Wechat.Bucket = "bk" }},
		{ProviderAWS, func(c *Config) { c.AWS.Region = "us-east-1"; c.AWS.AccessKeyID = "a"; c.AWS.SecretAccessKey = "b"; c.AWS.Bucket = "bk" }},
		{ProviderAzure, func(c *Config) { c.Azure.AccountName = "acct"; c.Azure.AccountKey = "key"; c.Azure.Container = "ctr" }},
	}
	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validTestConfig(tc.name)
			tc.cfg(&cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConfig_Validate_errors(t *testing.T) {
	cfg := ConfigDefault()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
	cfg.Provider.Name = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestConfigDefault_multipart(t *testing.T) {
	c := ConfigDefault()
	if c.Multipart.PartSize != defaultPartSize {
		t.Fatal(c.Multipart)
	}
}

func TestCOSConfig_regionOrDefault(t *testing.T) {
	c := COSConfig{}
	if c.regionOrDefault() != defaultTencentRegion {
		t.Fatal(c.regionOrDefault())
	}
}

func TestConfig_Validate_customRetry(t *testing.T) {
	cfg := validTestConfig(ProviderAliyun)
	cfg.Retry = RetryConfig{MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: time.Second, Multiplier: 2}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
