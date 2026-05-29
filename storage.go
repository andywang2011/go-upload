package upload

import (
	"context"
	"fmt"
	"io"
)

type storage struct {
	backend objectBackend
	cfg     Config
}

var _ Storage = (*storage)(nil)

func newStorage(cfg Config, backend objectBackend) *storage {
	return &storage{backend: backend, cfg: cfg}
}

// NewStorage creates a storage client with optional startup ping retry.
func NewStorage(cfg Config) (Storage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	backend, err := buildBackend(cfg)
	if err != nil {
		return nil, err
	}
	rm := newRetryManager(cfg.Retry)
	if err := rm.execute(context.Background(), backend.Ping); err != nil {
		_ = backend.Close()
		return nil, fmt.Errorf("upload ping: %w", err)
	}
	return newStorage(cfg, backend), nil
}

func (s *storage) Upload(ctx context.Context, key string, r io.Reader, opts ...PutOption) error {
	if s == nil || s.backend == nil {
		return ErrNotInitialized
	}
	o := applyPutOptions(opts)
	size := o.contentLength
	if o.useMultipart || (size > 0 && size >= s.cfg.Multipart.Threshold) {
		return uploadInParts(ctx, s.backend, key, r, size, s.cfg.Multipart)
	}
	return s.backend.Put(ctx, key, r, size, o.contentType)
}

func (s *storage) UploadLarge(ctx context.Context, key string, r io.Reader, size int64, opts ...PutOption) error {
	if s == nil || s.backend == nil {
		return ErrNotInitialized
	}
	o := applyPutOptions(opts)
	if size <= 0 {
		size = o.contentLength
	}
	return uploadInParts(ctx, s.backend, key, r, size, s.cfg.Multipart)
}

func (s *storage) Download(ctx context.Context, key string, w io.Writer, opts ...GetOption) error {
	if s == nil || s.backend == nil {
		return ErrNotInitialized
	}
	o := applyGetOptions(opts)
	size, err := s.backend.Head(ctx, key)
	if err != nil {
		return err
	}
	if o.useMultipart || size >= s.cfg.Multipart.Threshold {
		return downloadInParts(ctx, s.backend, key, w, size, s.cfg.Multipart)
	}
	return s.backend.Get(ctx, key, w)
}

func (s *storage) DownloadLarge(ctx context.Context, key string, w io.Writer, size int64, opts ...GetOption) error {
	if s == nil || s.backend == nil {
		return ErrNotInitialized
	}
	if size <= 0 {
		var err error
		size, err = s.backend.Head(ctx, key)
		if err != nil {
			return err
		}
	}
	return downloadInParts(ctx, s.backend, key, w, size, s.cfg.Multipart)
}

func (s *storage) List(ctx context.Context, prefix string, opts ...ListOption) ([]ObjectInfo, *ListPage, error) {
	if s == nil || s.backend == nil {
		return nil, nil, ErrNotInitialized
	}
	return s.backend.List(ctx, prefix, applyListOptions(opts))
}

func (s *storage) Delete(ctx context.Context, key string) error {
	if s == nil || s.backend == nil {
		return ErrNotInitialized
	}
	return s.backend.Delete(ctx, key)
}

func (s *storage) PublicURL(key string) string {
	if s == nil || s.backend == nil {
		return ""
	}
	return s.backend.PublicURL(key)
}

func (s *storage) Close() error {
	if s == nil || s.backend == nil {
		return nil
	}
	return s.backend.Close()
}
