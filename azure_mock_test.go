package upload

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

type mockAzureBlob struct {
	data map[string][]byte
	key  string
}

func (m *mockAzureBlob) UploadStream(ctx context.Context, body io.Reader, o *blockblob.UploadStreamOptions) (blockblob.UploadStreamResponse, error) {
	b, _ := io.ReadAll(body)
	if m.data == nil {
		m.data = map[string][]byte{}
	}
	m.data[m.keyName()] = b
	return blockblob.UploadStreamResponse{}, nil
}

func (m *mockAzureBlob) DownloadStream(ctx context.Context, o *blob.DownloadStreamOptions) (blob.DownloadStreamResponse, error) {
	return blob.DownloadStreamResponse{}, io.EOF
}

func (m *mockAzureBlob) GetProperties(ctx context.Context, o *blob.GetPropertiesOptions) (blob.GetPropertiesResponse, error) {
	n := int64(len(m.data[m.keyName()]))
	return blob.GetPropertiesResponse{ContentLength: &n}, nil
}

func (m *mockAzureBlob) Delete(ctx context.Context, o *blob.DeleteOptions) (blob.DeleteResponse, error) {
	delete(m.data, m.keyName())
	return blob.DeleteResponse{}, nil
}

func (m *mockAzureBlob) StageBlock(ctx context.Context, base64BlockID string, body io.ReadSeekCloser, options *blockblob.StageBlockOptions) (blockblob.StageBlockResponse, error) {
	_, _ = io.ReadAll(body)
	return blockblob.StageBlockResponse{}, nil
}

func (m *mockAzureBlob) CommitBlockList(ctx context.Context, base64BlockIDs []string, options *blockblob.CommitBlockListOptions) (blockblob.CommitBlockListResponse, error) {
	return blockblob.CommitBlockListResponse{}, nil
}

func (m *mockAzureBlob) keyName() string {
	if m.key != "" {
		return m.key
	}
	return "obj"
}
