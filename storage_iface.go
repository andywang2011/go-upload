package upload

import (
	"context"
	"io"
)

// ObjectInfo describes one listed object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified int64 // unix seconds when unknown
	ETag         string
}

// ListPage carries pagination for List.
type ListPage struct {
	NextContinuationToken string
	IsTruncated           bool
}

// Storage is the primary object storage abstraction.
type Storage interface {
	Upload(ctx context.Context, key string, r io.Reader, opts ...PutOption) error
	Download(ctx context.Context, key string, w io.Writer, opts ...GetOption) error
	UploadLarge(ctx context.Context, key string, r io.Reader, size int64, opts ...PutOption) error
	DownloadLarge(ctx context.Context, key string, w io.Writer, size int64, opts ...GetOption) error
	List(ctx context.Context, prefix string, opts ...ListOption) ([]ObjectInfo, *ListPage, error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
	Close() error
}
