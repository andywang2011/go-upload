package upload

import (
	"bytes"
	"context"
	"testing"
)

func TestAzureDefaultOps_withMockBlob(t *testing.T) {
	mock := &mockAzureBlob{data: map[string][]byte{}}
	b := &azureBackend{container: "ctr", cfg: AzureConfig{}, mp: MultipartConfig{PartSize: 4}}
	restore := saveAzureHooks()
	defer restore()
	newAzureBlockBlob = func(ab *azureBackend, key string) blockBlobAPI {
		mock.key = key
		return mock
	}
	ctx := context.Background()
	if err := azurePut(b, ctx, "obj", bytes.NewReader([]byte("x")), 1, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if _, err := azureHead(b, ctx, "obj"); err != nil {
		t.Fatal(err)
	}
	if err := azureUploadMultipart(b, ctx, "obj", bytes.NewReader([]byte("12345678")), 8, b.mp); err != nil {
		t.Fatal(err)
	}
	if err := azureDelete(b, ctx, "obj"); err != nil {
		t.Fatal(err)
	}
}
