package upload

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

// blockBlobAPI is the block blob surface used by azureBackend (mockable in tests).
type blockBlobAPI interface {
	UploadStream(ctx context.Context, body io.Reader, o *blockblob.UploadStreamOptions) (blockblob.UploadStreamResponse, error)
	DownloadStream(ctx context.Context, o *blob.DownloadStreamOptions) (blob.DownloadStreamResponse, error)
	GetProperties(ctx context.Context, o *blob.GetPropertiesOptions) (blob.GetPropertiesResponse, error)
	Delete(ctx context.Context, o *blob.DeleteOptions) (blob.DeleteResponse, error)
	StageBlock(ctx context.Context, base64BlockID string, body io.ReadSeekCloser, options *blockblob.StageBlockOptions) (blockblob.StageBlockResponse, error)
	CommitBlockList(ctx context.Context, base64BlockIDs []string, options *blockblob.CommitBlockListOptions) (blockblob.CommitBlockListResponse, error)
}

var newAzureBlockBlob = func(b *azureBackend, key string) blockBlobAPI {
	return b.client.ServiceClient().NewContainerClient(b.container).NewBlockBlobClient(key)
}

var azurePing = func(b *azureBackend, ctx context.Context) error {
	pager := b.client.NewListBlobsFlatPager(b.container, &azblob.ListBlobsFlatOptions{
		MaxResults: int32Ptr(1),
	})
	_, err := pager.NextPage(ctx)
	return err
}
