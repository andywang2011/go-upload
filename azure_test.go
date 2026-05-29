package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func TestAzureNotFound(t *testing.T) {
	if azureNotFound(nil) {
		t.Fatal()
	}
	err := &azcore.ResponseError{StatusCode: 404}
	if !azureNotFound(err) {
		t.Fatal()
	}
}

func TestNewAzureBackend_hook(t *testing.T) {
	old := newAzureClient
	newAzureClient = func(cfg AzureConfig) (*azblob.Client, error) {
		return nil, ErrInvalidConfig
	}
	defer func() { newAzureClient = old }()
	_, err := newAzureBackend(AzureConfig{AccountName: "a", AccountKey: "k", Container: "c"}, ConfigDefault().Multipart)
	if err == nil {
		t.Fatal()
	}
}

func TestAzureBackend_publicURL(t *testing.T) {
	b := &azureBackend{
		container: "ctr",
		cfg:       AzureConfig{CDNHost: "https://azure.cdn"},
		mp:        ConfigDefault().Multipart,
	}
	if b.PublicURL("k") != "https://azure.cdn/k" {
		t.Fatal(b.PublicURL("k"))
	}
	b2 := &azureBackend{container: "ctr", cfg: AzureConfig{}, mp: ConfigDefault().Multipart}
	if b2.PublicURL("k") == "" {
		t.Fatal()
	}
	if mapAzureErr(errors.New("BlobNotFound")) != ErrNotFound {
		t.Fatal()
	}
	_ = b.Close()
}

func TestInt32Ptr(t *testing.T) {
	if *int32Ptr(3) != 3 {
		t.Fatal()
	}
}

func TestAzureContainerURL(t *testing.T) {
	if azureContainerURL(nil, "c") != "" {
		t.Fatal()
	}
}

func TestAzureBackend_mockBlob(t *testing.T) {
	mock := &mockAzureBlob{data: map[string][]byte{"obj": []byte("azure-data")}}
	b := &azureBackend{
		container: "ctr",
		cfg:       AzureConfig{CDNHost: "https://azure.cdn"},
		mp:        MultipartConfig{PartSize: 4},
	}
	restore := saveAzureHooks()
	defer restore()
	newAzureBlockBlob = func(ab *azureBackend, key string) blockBlobAPI {
		mock.key = key
		return mock
	}
	azurePing = func(ab *azureBackend, ctx context.Context) error { return nil }
	azureGet = func(ab *azureBackend, ctx context.Context, key string, w io.Writer) error {
		_, err := w.Write(mock.data[key])
		return err
	}
	azureGetRange = func(ab *azureBackend, ctx context.Context, key string, w io.Writer, offset, length int64) error {
		data := mock.data[key]
		if offset >= int64(len(data)) {
			return io.EOF
		}
		end := offset + length
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		_, err := w.Write(data[offset:end])
		return err
	}
	azureListBlobs = func(ab *azureBackend, ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
		return []ObjectInfo{{Key: "obj", Size: int64(len(mock.data["obj"]))}}, &ListPage{}, nil
	}
	ctx := context.Background()
	if err := b.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Put(ctx, "obj", bytes.NewReader([]byte("new")), 3, "text/plain"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := b.Get(ctx, "obj", &buf); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Head(ctx, "obj"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.List(ctx, "", listOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := b.UploadMultipart(ctx, "obj", bytes.NewReader([]byte("12345678")), 8, b.mp); err != nil {
		t.Fatal(err)
	}
	mock.data["obj"] = []byte("12345678")
	var dl bytes.Buffer
	if err := b.DownloadMultipart(ctx, "obj", &dl, 8, b.mp); err != nil {
		t.Fatal(err)
	}
	if err := b.Delete(ctx, "obj"); err != nil {
		t.Fatal(err)
	}
}

func saveAzureHooks() func() {
	s := struct {
		blob func(*azureBackend, string) blockBlobAPI
		ping func(*azureBackend, context.Context) error
		get  func(*azureBackend, context.Context, string, io.Writer) error
		gr   func(*azureBackend, context.Context, string, io.Writer, int64, int64) error
		list func(*azureBackend, context.Context, string, listOptions) ([]ObjectInfo, *ListPage, error)
		put  func(*azureBackend, context.Context, string, io.Reader, int64, string) error
		del  func(*azureBackend, context.Context, string) error
		head func(*azureBackend, context.Context, string) (int64, error)
		ump  func(*azureBackend, context.Context, string, io.Reader, int64, MultipartConfig) error
		dmp  func(*azureBackend, context.Context, string, io.Writer, int64, MultipartConfig) error
	}{
		blob: newAzureBlockBlob, ping: azurePing, get: azureGet, gr: azureGetRange, list: azureListBlobs,
		put: azurePut, del: azureDelete, head: azureHead, ump: azureUploadMultipart, dmp: azureDownloadMultipart,
	}
	return func() {
		newAzureBlockBlob, azurePing, azureGet, azureGetRange, azureListBlobs = s.blob, s.ping, s.get, s.gr, s.list
		azurePut, azureDelete, azureHead, azureUploadMultipart, azureDownloadMultipart = s.put, s.del, s.head, s.ump, s.dmp
	}
}
