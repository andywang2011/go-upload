package upload

import (
	"context"
	"io"
)

// objectBackend is the internal driver contract.
type objectBackend interface {
	Ping(ctx context.Context) error
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string, w io.Writer) error
	GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error
	List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error)
	Delete(ctx context.Context, key string) error
	UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error
	DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error
	Head(ctx context.Context, key string) (int64, error)
	PublicURL(key string) string
	Close() error
}
