package upload

import (
	"bytes"
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func TestBuildBackend_allProviders(t *testing.T) {
	cases := []struct {
		provider string
		setup    func()
	}{
		{ProviderAliyun, func() {
			newOSSBucket = func(AliyunConfig) (ossBucketAPI, error) {
				return &mockOSSBucket{objects: map[string][]byte{}}, nil
			}
		}},
		{ProviderTencent, func() {
			newCOSClient = func(COSConfig) (cosAPI, error) { return &mockCOS{objects: map[string][]byte{}}, nil }
		}},
		{ProviderWechat, func() {
			newCOSClient = func(COSConfig) (cosAPI, error) { return &mockCOS{objects: map[string][]byte{}}, nil }
		}},
		{ProviderAWS, func() {
			newAWSClient = func(AWSConfig) (awsAPI, error) { return &mockAWS{objects: map[string][]byte{}}, nil }
		}},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			restore := saveHooks()
			defer restore()
			tc.setup()
			cfg := validTestConfig(tc.provider)
			b, err := buildBackend(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.Ping(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := b.Put(context.Background(), "obj", bytes.NewReader([]byte("x")), 1, ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDefaultBuildBackend_azure(t *testing.T) {
	restore := saveHooks()
	defer restore()
	newAzureClient = func(AzureConfig) (*azblob.Client, error) { return &azblob.Client{}, nil }
	azurePing = func(*azureBackend, context.Context) error { return nil }
	newAzureBlockBlob = func(b *azureBackend, key string) blockBlobAPI {
		return &mockAzureBlob{data: map[string][]byte{}, key: key}
	}
	cfg := validTestConfig(ProviderAzure)
	b, err := buildBackend(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Put(context.Background(), "k", bytes.NewReader([]byte("1")), 1, ""); err != nil {
		t.Fatal(err)
	}
}

func saveHooks() func() {
	ossH, cosH, awsH := newOSSBucket, newCOSClient, newAWSClient
	azH, pingH, blobH := newAzureClient, azurePing, newAzureBlockBlob
	return func() {
		newOSSBucket, newCOSClient, newAWSClient = ossH, cosH, awsH
		newAzureClient, azurePing, newAzureBlockBlob = azH, pingH, blobH
	}
}
