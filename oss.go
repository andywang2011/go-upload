// Aliyun OSS driver (provider: aliyun).
package upload

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type ossBackend struct {
	bucket ossBucketAPI
	cfg    AliyunConfig
	mp     MultipartConfig
}

func newOSSBackend(cfg AliyunConfig, mp MultipartConfig) (objectBackend, error) {
	bucket, err := newOSSBucket(cfg)
	if err != nil {
		return nil, err
	}
	return &ossBackend{bucket: bucket, cfg: cfg, mp: mp}, nil
}

func (b *ossBackend) Ping(ctx context.Context) error {
	_, err := b.bucket.ListObjectsV2(oss.MaxKeys(1))
	_ = ctx
	return err
}

func (b *ossBackend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := ossPutOptions(ctx, size, contentType)
	return b.bucket.PutObject(key, r, opts...)
}

func (b *ossBackend) Get(ctx context.Context, key string, w io.Writer) error {
	body, err := b.bucket.GetObject(key, oss.WithContext(ctx))
	if err != nil {
		return mapOSSErr(err)
	}
	defer body.Close()
	_, err = io.Copy(w, body)
	return err
}

func (b *ossBackend) GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error {
	end, err := validateRange(offset, length)
	if err != nil {
		return err
	}
	body, err := b.bucket.GetObject(key, oss.WithContext(ctx), oss.Range(offset, end))
	if err != nil {
		return mapOSSErr(err)
	}
	defer body.Close()
	_, err = io.Copy(w, body)
	return err
}

func (b *ossBackend) List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	ossOpts := []oss.Option{oss.WithContext(ctx), oss.Prefix(prefix)}
	if opts.maxKeys > 0 {
		ossOpts = append(ossOpts, oss.MaxKeys(opts.maxKeys))
	}
	if opts.continuationToken != "" {
		ossOpts = append(ossOpts, oss.ContinuationToken(opts.continuationToken))
	}
	if opts.delimiter != "" {
		ossOpts = append(ossOpts, oss.Delimiter(opts.delimiter))
	}
	res, err := b.bucket.ListObjectsV2(ossOpts...)
	if err != nil {
		return nil, nil, err
	}
	out := make([]ObjectInfo, 0, len(res.Objects))
	for _, o := range res.Objects {
		out = append(out, ObjectInfo{
			Key:          o.Key,
			Size:         o.Size,
			LastModified: o.LastModified.Unix(),
			ETag:         strings.Trim(o.ETag, "\""),
		})
	}
	page := &ListPage{
		NextContinuationToken: res.NextContinuationToken,
		IsTruncated:           res.IsTruncated,
	}
	return out, page, nil
}

func (b *ossBackend) Delete(ctx context.Context, key string) error {
	return b.bucket.DeleteObject(key, oss.WithContext(ctx))
}

func (b *ossBackend) Head(ctx context.Context, key string) (int64, error) {
	h, err := b.bucket.GetObjectMeta(key, oss.WithContext(ctx))
	if err != nil {
		return 0, mapOSSErr(err)
	}
	cl := h.Get("Content-Length")
	if cl == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (b *ossBackend) UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	imur, err := b.bucket.InitiateMultipartUpload(key, oss.WithContext(ctx))
	if err != nil {
		return err
	}
	partSize := mp.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	var parts []oss.UploadPart
	partNum := 1
	remaining := size
	for {
		if ctx.Err() != nil {
			return joinAbortErr(ctx.Err(), b.bucket.AbortMultipartUpload(imur))
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
		up, err := b.bucket.UploadPart(imur, bytes.NewReader(buf), int64(len(buf)), partNum)
		if err != nil {
			return joinAbortErr(err, b.bucket.AbortMultipartUpload(imur))
		}
		parts = append(parts, up)
		partNum++
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
			return joinAbortErr(readErr, b.bucket.AbortMultipartUpload(imur))
		}
	}
	_, err = b.bucket.CompleteMultipartUpload(imur, parts, oss.WithContext(ctx))
	return err
}

func (b *ossBackend) DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
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

func (b *ossBackend) PublicURL(key string) string {
	return PublicURLFromHost(b.cfg.CDNHost, key)
}

func (b *ossBackend) Close() error { return nil }

func ossPutOptions(ctx context.Context, size int64, contentType string) []oss.Option {
	opts := []oss.Option{oss.WithContext(ctx)}
	if size > 0 {
		opts = append(opts, oss.ContentLength(size))
	}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	return opts
}

func mapOSSErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "404") {
		return ErrNotFound
	}
	return err
}
