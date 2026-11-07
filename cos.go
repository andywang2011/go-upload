// Tencent COS / WeChat COS driver (providers: tencent, wechat).
package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	cos "github.com/tencentyun/cos-go-sdk-v5"
)

// cosAPI is the subset of COS SDK calls used by the driver (mockable in tests).
type cosAPI interface {
	BucketGet(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error)
	ObjectPut(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	ObjectGet(ctx context.Context, name string, opt *cos.ObjectGetOptions) (*cos.Response, error)
	ObjectHead(ctx context.Context, name string, opt *cos.ObjectHeadOptions) (*cos.Response, error)
	ObjectDelete(ctx context.Context, name string) (*cos.Response, error)
	ObjectInitiateMultipartUpload(ctx context.Context, name string, opt *cos.InitiateMultipartUploadOptions) (*cos.InitiateMultipartUploadResult, *cos.Response, error)
	ObjectUploadPart(ctx context.Context, name, uploadID string, partNumber int, r io.Reader, opt *cos.ObjectUploadPartOptions) (*cos.Response, error)
	ObjectCompleteMultipartUpload(ctx context.Context, name, uploadID string, opt *cos.CompleteMultipartUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error)
	ObjectAbortMultipartUpload(ctx context.Context, name, uploadID string) (*cos.Response, error)
	BucketBaseURL() string
}

type cosClientAdapter struct{ *cos.Client }

func (a *cosClientAdapter) BucketGet(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error) {
	return a.Bucket.Get(ctx, opt)
}
func (a *cosClientAdapter) ObjectPut(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	return a.Object.Put(ctx, name, r, opt)
}
func (a *cosClientAdapter) ObjectGet(ctx context.Context, name string, opt *cos.ObjectGetOptions) (*cos.Response, error) {
	return a.Object.Get(ctx, name, opt)
}
func (a *cosClientAdapter) ObjectHead(ctx context.Context, name string, opt *cos.ObjectHeadOptions) (*cos.Response, error) {
	return a.Object.Head(ctx, name, opt)
}
func (a *cosClientAdapter) ObjectDelete(ctx context.Context, name string) (*cos.Response, error) {
	return a.Object.Delete(ctx, name)
}
func (a *cosClientAdapter) ObjectInitiateMultipartUpload(ctx context.Context, name string, opt *cos.InitiateMultipartUploadOptions) (*cos.InitiateMultipartUploadResult, *cos.Response, error) {
	return a.Object.InitiateMultipartUpload(ctx, name, opt)
}
func (a *cosClientAdapter) ObjectUploadPart(ctx context.Context, name, uploadID string, partNumber int, r io.Reader, opt *cos.ObjectUploadPartOptions) (*cos.Response, error) {
	return a.Object.UploadPart(ctx, name, uploadID, partNumber, r, opt)
}
func (a *cosClientAdapter) ObjectCompleteMultipartUpload(ctx context.Context, name, uploadID string, opt *cos.CompleteMultipartUploadOptions) (*cos.CompleteMultipartUploadResult, *cos.Response, error) {
	return a.Object.CompleteMultipartUpload(ctx, name, uploadID, opt)
}
func (a *cosClientAdapter) ObjectAbortMultipartUpload(ctx context.Context, name, uploadID string) (*cos.Response, error) {
	return a.Object.AbortMultipartUpload(ctx, name, uploadID)
}
func (a *cosClientAdapter) BucketBaseURL() string {
	if a.BaseURL != nil && a.BaseURL.BucketURL != nil {
		return a.BaseURL.BucketURL.String()
	}
	return ""
}

type cosBackend struct {
	client cosAPI
	cfg    COSConfig
	mp     MultipartConfig
}

func newCOSBackend(cfg COSConfig, mp MultipartConfig) (objectBackend, error) {
	client, err := newCOSClient(cfg)
	if err != nil {
		return nil, err
	}
	return &cosBackend{client: client, cfg: cfg, mp: mp}, nil
}

func (b *cosBackend) Ping(ctx context.Context) error {
	_, _, err := b.client.BucketGet(ctx, nil)
	return err
}

func (b *cosBackend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: contentType,
		},
	}
	if size > 0 {
		opt.ObjectPutHeaderOptions.ContentLength = size
	}
	_, err := b.client.ObjectPut(ctx, key, r, opt)
	return mapCOSErr(err)
}

func (b *cosBackend) Get(ctx context.Context, key string, w io.Writer) error {
	resp, err := b.client.ObjectGet(ctx, key, nil)
	if err != nil {
		return mapCOSErr(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

func (b *cosBackend) GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error {
	end, err := validateRange(offset, length)
	if err != nil {
		return err
	}
	opt := &cos.ObjectGetOptions{
		Range: fmt.Sprintf("bytes=%d-%d", offset, end),
	}
	resp, err := b.client.ObjectGet(ctx, key, opt)
	if err != nil {
		return mapCOSErr(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

func (b *cosBackend) List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	opt := &cos.BucketGetOptions{
		Prefix: prefix,
	}
	if opts.maxKeys > 0 {
		opt.MaxKeys = opts.maxKeys
	}
	if opts.continuationToken != "" {
		opt.Marker = opts.continuationToken
	}
	if opts.delimiter != "" {
		opt.Delimiter = opts.delimiter
	}
	res, _, err := b.client.BucketGet(ctx, opt)
	if err != nil {
		return nil, nil, err
	}
	out := make([]ObjectInfo, 0, len(res.Contents))
	for _, o := range res.Contents {
		out = append(out, ObjectInfo{
			Key:  o.Key,
			Size: o.Size,
			ETag: strings.Trim(o.ETag, "\""),
		})
	}
	page := &ListPage{
		NextContinuationToken: res.NextMarker,
		IsTruncated:           res.IsTruncated,
	}
	return out, page, nil
}

func (b *cosBackend) Delete(ctx context.Context, key string) error {
	_, err := b.client.ObjectDelete(ctx, key)
	return mapCOSErr(err)
}

func (b *cosBackend) Head(ctx context.Context, key string) (int64, error) {
	resp, err := b.client.ObjectHead(ctx, key, nil)
	if err != nil {
		return 0, mapCOSErr(err)
	}
	if resp == nil || resp.Header == nil {
		return 0, nil
	}
	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		return 0, nil
	}
	return strconv.ParseInt(cl, 10, 64)
}

func (b *cosBackend) UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	v, _, err := b.client.ObjectInitiateMultipartUpload(ctx, key, nil)
	if err != nil {
		return err
	}
	uploadID := v.UploadID
	partSize := mp.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	partNum := 1
	remaining := size
	var completeParts []cos.Object
	for {
		if ctx.Err() != nil {
			return joinAbortErr(ctx.Err(), b.abortUpload(ctx, key, uploadID))
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
		resp, err := b.client.ObjectUploadPart(ctx, key, uploadID, partNum, bytes.NewReader(buf), nil)
		if err != nil {
			return joinAbortErr(err, b.abortUpload(ctx, key, uploadID))
		}
		etag := ""
		if resp != nil && resp.Header != nil {
			etag = resp.Header.Get("ETag")
		}
		completeParts = append(completeParts, cos.Object{PartNumber: partNum, ETag: etag})
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
			return joinAbortErr(readErr, b.abortUpload(ctx, key, uploadID))
		}
	}
	_, _, err = b.client.ObjectCompleteMultipartUpload(ctx, key, uploadID, &cos.CompleteMultipartUploadOptions{
		Parts: completeParts,
	})
	return err
}

// abortUpload aborts a multipart upload using a context that survives ctx cancellation.
func (b *cosBackend) abortUpload(ctx context.Context, key, uploadID string) error {
	actx, cancel := abortContext(ctx)
	defer cancel()
	_, err := b.client.ObjectAbortMultipartUpload(actx, key, uploadID)
	return err
}

func (b *cosBackend) DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
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

func (b *cosBackend) PublicURL(key string) string {
	host := b.cfg.CDNHost
	if host == "" && b.client != nil {
		host = b.client.BucketBaseURL()
	}
	return PublicURLFromHost(host, key)
}

func (b *cosBackend) Close() error { return nil }

func mapCOSErr(err error) error {
	if err == nil {
		return nil
	}
	var resp *cos.ErrorResponse
	if errors.As(err, &resp) && resp != nil && resp.Response != nil && resp.Response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	msg := err.Error()
	if strings.Contains(msg, "404") || strings.Contains(msg, "NoSuchKey") {
		return ErrNotFound
	}
	return err
}
