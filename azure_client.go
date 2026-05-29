package upload

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func newAzureClientFromConfig(cfg AzureConfig) (*azblob.Client, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.AccountName)
	}
	cred, err := azblob.NewSharedKeyCredential(cfg.AccountName, cfg.AccountKey)
	if err != nil {
		return nil, err
	}
	return azblob.NewClientWithSharedKeyCredential(endpoint, cred, nil)
}

func defaultNewAzureClient(cfg AzureConfig) (*azblob.Client, error) {
	return newAzureClientFromConfig(cfg)
}

func azureContainerURL(client *azblob.Client, container string) string {
	if client == nil {
		return ""
	}
	// Service URL ends with / ; container path is appended by SDK helpers.
	return strings.TrimSuffix(fmt.Sprint(client.URL()), "/") + "/" + container
}

// azureNotFound checks Azure blob not-found errors.
func azureNotFound(err error) bool {
	if err == nil {
		return false
	}
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode == 404 {
		return true
	}
	return strings.Contains(err.Error(), "BlobNotFound") || strings.Contains(err.Error(), "404")
}
