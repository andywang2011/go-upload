package upload

import (
	"context"
	"io"
	"strings"
)

// Uploader is the legacy upload-only interface used by demo-api.
type Uploader interface {
	Upload(ctx context.Context, path string, reader io.Reader) error
	CdnUrl(path string) string
	BaseURL() string
}

// uploaderAdapter adapts Storage to Uploader.
type uploaderAdapter struct {
	s Storage
}

// AsUploader returns a Uploader view of Storage.
func AsUploader(s Storage) Uploader {
	if s == nil {
		return nil
	}
	return &uploaderAdapter{s: s}
}

func (a *uploaderAdapter) Upload(ctx context.Context, path string, reader io.Reader) error {
	return a.s.Upload(ctx, path, reader)
}

func (a *uploaderAdapter) CdnUrl(path string) string {
	return a.s.PublicURL(path)
}

func (a *uploaderAdapter) BaseURL() string {
	return strings.TrimSuffix(a.s.PublicURL(""), "/")
}

// PublicURLFromHost joins CDN host and object key.
func PublicURLFromHost(host, key string) string {
	host = strings.TrimSuffix(host, "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return host
	}
	if host == "" {
		return "/" + key
	}
	return host + "/" + key
}
