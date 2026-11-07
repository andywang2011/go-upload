package upload

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestDefaultNewOSSBucket(t *testing.T) {
	_, err := defaultNewOSSBucket(AliyunConfig{
		Endpoint: "oss-cn-hangzhou.aliyuncs.com", AccessKeyID: "id", AccessKeySecret: "sec", Bucket: "mybucket",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefaultNewCOSClient(t *testing.T) {
	api, err := defaultNewCOSClient(COSConfig{
		SecretID: "id", SecretKey: "sec", Bucket: "bk", Region: "ap-shanghai",
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.BucketBaseURL() == "" {
		t.Fatal("expected bucket url")
	}
	_, _, _ = api.BucketGet(context.Background(), nil)
}

func TestDefaultNewAWSClient(t *testing.T) {
	_, err := defaultNewAWSClient(AWSConfig{
		Region: "us-east-1", AccessKeyID: "a", SecretAccessKey: "b", Bucket: "bk",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewAzureClientFromConfig_validKey(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	_, err := newAzureClientFromConfig(AzureConfig{AccountName: "acct", AccountKey: key})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefaultNewAzureClient(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	_, err := defaultNewAzureClient(AzureConfig{AccountName: "acct", AccountKey: key, Container: "c"})
	if err != nil {
		t.Fatal(err)
	}
}
