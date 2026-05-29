// Microsoft Azure Blob driver (provider: azure).
package upload

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

type azureBackend struct {
	client    *azblob.Client
	container string
	cfg       AzureConfig
	mp        MultipartConfig
}

func newAzureBackend(cfg AzureConfig, mp MultipartConfig) (objectBackend, error) {
	client, err := newAzureClient(cfg)
	if err != nil {
		return nil, err
	}
	return &azureBackend{
		client:    client,
		container: cfg.Container,
		cfg:       cfg,
		mp:        mp,
	}, nil
}

func (b *azureBackend) blobClient(key string) blockBlobAPI {
	return newAzureBlockBlob(b, key)
}

func (b *azureBackend) Ping(ctx context.Context) error {
	return azurePing(b, ctx)
}

func (b *azureBackend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	return azurePut(b, ctx, key, r, size, contentType)
}

var azurePut = func(b *azureBackend, ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := &blockblob.UploadStreamOptions{}
	if contentType != "" {
		opts.HTTPHeaders = &blob.HTTPHeaders{BlobContentType: &contentType}
	}
	_, err := b.blobClient(key).UploadStream(ctx, r, opts)
	return mapAzureErr(err)
}

func (b *azureBackend) Get(ctx context.Context, key string, w io.Writer) error {
	return azureGet(b, ctx, key, w)
}

var azureGet = func(b *azureBackend, ctx context.Context, key string, w io.Writer) error {
	resp, err := b.blobClient(key).DownloadStream(ctx, nil)
	if err != nil {
		return mapAzureErr(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

func (b *azureBackend) GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error {
	return azureGetRange(b, ctx, key, w, offset, length)
}

var azureGetRange = func(b *azureBackend, ctx context.Context, key string, w io.Writer, offset, length int64) error {
	resp, err := b.blobClient(key).DownloadStream(ctx, &blob.DownloadStreamOptions{
		Range: blob.HTTPRange{Offset: offset, Count: length},
	})
	if err != nil {
		return mapAzureErr(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

func (b *azureBackend) List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	return azureListBlobs(b, ctx, prefix, opts)
}

var azureListBlobs = func(b *azureBackend, ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	listOpts := &azblob.ListBlobsFlatOptions{Prefix: &prefix}
	if opts.maxKeys > 0 {
		listOpts.MaxResults = int32Ptr(int32(opts.maxKeys))
	}
	pager := b.client.NewListBlobsFlatPager(b.container, listOpts)
	page, err := pager.NextPage(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]ObjectInfo, 0, len(page.Segment.BlobItems))
	for _, item := range page.Segment.BlobItems {
		if item.Name == nil {
			continue
		}
		var size int64
		if item.Properties != nil && item.Properties.ContentLength != nil {
			size = *item.Properties.ContentLength
		}
		var mod int64
		if item.Properties != nil && item.Properties.LastModified != nil {
			mod = item.Properties.LastModified.Unix()
		}
		etag := ""
		if item.Properties != nil && item.Properties.ETag != nil {
			etag = string(*item.Properties.ETag)
		}
		out = append(out, ObjectInfo{Key: *item.Name, Size: size, LastModified: mod, ETag: etag})
	}
	lp := &ListPage{IsTruncated: page.NextMarker != nil}
	if page.NextMarker != nil {
		lp.NextContinuationToken = *page.NextMarker
	}
	return out, lp, nil
}

func (b *azureBackend) Delete(ctx context.Context, key string) error {
	return azureDelete(b, ctx, key)
}

var azureDelete = func(b *azureBackend, ctx context.Context, key string) error {
	_, err := b.blobClient(key).Delete(ctx, nil)
	return mapAzureErr(err)
}

func (b *azureBackend) Head(ctx context.Context, key string) (int64, error) {
	return azureHead(b, ctx, key)
}

var azureHead = func(b *azureBackend, ctx context.Context, key string) (int64, error) {
	resp, err := b.blobClient(key).GetProperties(ctx, nil)
	if err != nil {
		return 0, mapAzureErr(err)
	}
	if resp.ContentLength == nil {
		return 0, nil
	}
	return *resp.ContentLength, nil
}

func (b *azureBackend) UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	return azureUploadMultipart(b, ctx, key, r, size, mp)
}

var azureUploadMultipart = func(b *azureBackend, ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	partSize := mp.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	bc := b.blobClient(key)
	blockIDs := make([]string, 0)
	remaining := size
	partIdx := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n := partSize
		if size > 0 && remaining < n {
			n = remaining
		}
		buf := make([]byte, n)
		got, readErr := io.ReadFull(r, buf)
		if got == 0 && readErr == io.EOF {
			break
		}
		if got < len(buf) {
			buf = buf[:got]
		}
		blockID := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%05d", partIdx)))
		_, err := bc.StageBlock(ctx, blockID, streaming.NopCloser(bytes.NewReader(buf)), nil)
		if err != nil {
			return err
		}
		blockIDs = append(blockIDs, blockID)
		partIdx++
		if size > 0 {
			remaining -= int64(got)
			if remaining <= 0 {
				break
			}
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	_, err := bc.CommitBlockList(ctx, blockIDs, nil)
	return err
}

func (b *azureBackend) DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
	return azureDownloadMultipart(b, ctx, key, w, size, mp)
}

var azureDownloadMultipart = func(b *azureBackend, ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
	partSize := mp.DownloadPartSize
	if partSize <= 0 {
		partSize = mp.PartSize
	}
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	for off := int64(0); off < size; off += partSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n := partSize
		if off+n > size {
			n = size - off
		}
		if err := b.GetRange(ctx, key, w, off, n); err != nil {
			return err
		}
	}
	return nil
}

func (b *azureBackend) PublicURL(key string) string {
	host := b.cfg.CDNHost
	if host == "" {
		host = azureContainerURL(b.client, b.container)
	}
	return PublicURLFromHost(host, key)
}

func (b *azureBackend) Close() error { return nil }

func mapAzureErr(err error) error {
	if err == nil {
		return nil
	}
	if azureNotFound(err) {
		return ErrNotFound
	}
	return err
}

func int32Ptr(n int32) *int32 { return &n }
