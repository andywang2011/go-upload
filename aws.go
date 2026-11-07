// AWS S3 driver (provider: aws). Uses aws-sdk-go-v2/service/s3 under the hood.
package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// awsAPI is the subset of S3 client methods used by the AWS driver (mockable in tests).
type awsAPI interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

type awsBackend struct {
	client awsAPI
	cfg    AWSConfig
	mp     MultipartConfig
}

func newAWSBackend(cfg AWSConfig, mp MultipartConfig) (objectBackend, error) {
	client, err := newAWSClient(cfg)
	if err != nil {
		return nil, err
	}
	return &awsBackend{client: client, cfg: cfg, mp: mp}, nil
}

func (b *awsBackend) Ping(ctx context.Context) error {
	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(b.cfg.Bucket)})
	return err
}

func (b *awsBackend) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
		Body:   r,
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if size > 0 {
		in.ContentLength = aws.Int64(size)
	}
	_, err := b.client.PutObject(ctx, in)
	return mapAWSErr(err)
}

func (b *awsBackend) Get(ctx context.Context, key string, w io.Writer) error {
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return mapAWSErr(err)
	}
	defer out.Body.Close()
	_, err = io.Copy(w, out.Body)
	return err
}

func (b *awsBackend) GetRange(ctx context.Context, key string, w io.Writer, offset, length int64) error {
	end, err := validateRange(offset, length)
	if err != nil {
		return err
	}
	rng := fmt.Sprintf("bytes=%d-%d", offset, end)
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
		Range:  aws.String(rng),
	})
	if err != nil {
		return mapAWSErr(err)
	}
	defer out.Body.Close()
	_, err = io.Copy(w, out.Body)
	return err
}

func (b *awsBackend) List(ctx context.Context, prefix string, opts listOptions) ([]ObjectInfo, *ListPage, error) {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.cfg.Bucket),
		Prefix: aws.String(prefix),
	}
	if opts.maxKeys > 0 {
		in.MaxKeys = aws.Int32(int32(opts.maxKeys))
	}
	if opts.continuationToken != "" {
		in.ContinuationToken = aws.String(opts.continuationToken)
	}
	if opts.delimiter != "" {
		in.Delimiter = aws.String(opts.delimiter)
	}
	res, err := b.client.ListObjectsV2(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	out := make([]ObjectInfo, 0, len(res.Contents))
	for _, o := range res.Contents {
		var size int64
		if o.Size != nil {
			size = *o.Size
		}
		var mod int64
		if o.LastModified != nil {
			mod = o.LastModified.Unix()
		}
		etag := ""
		if o.ETag != nil {
			etag = strings.Trim(*o.ETag, "\"")
		}
		key := ""
		if o.Key != nil {
			key = *o.Key
		}
		out = append(out, ObjectInfo{Key: key, Size: size, LastModified: mod, ETag: etag})
	}
	page := &ListPage{}
	if res.IsTruncated != nil {
		page.IsTruncated = *res.IsTruncated
	}
	if res.NextContinuationToken != nil {
		page.NextContinuationToken = *res.NextContinuationToken
	}
	return out, page, nil
}

func (b *awsBackend) Delete(ctx context.Context, key string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	return mapAWSErr(err)
}

func (b *awsBackend) Head(ctx context.Context, key string) (int64, error) {
	out, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, mapAWSErr(err)
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (b *awsBackend) UploadMultipart(ctx context.Context, key string, r io.Reader, size int64, mp MultipartConfig) error {
	create, err := b.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	uploadID := create.UploadId
	partSize := mp.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	var completed []types.CompletedPart
	partNum := int32(1)
	remaining := size
	abort := func(cause error) error {
		actx, cancel := abortContext(ctx)
		defer cancel()
		_, abortErr := b.client.AbortMultipartUpload(actx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(b.cfg.Bucket),
			Key:      aws.String(key),
			UploadId: uploadID,
		})
		return joinAbortErr(cause, abortErr)
	}
	for {
		if ctx.Err() != nil {
			return abort(ctx.Err())
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
		up, err := b.client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(b.cfg.Bucket),
			Key:        aws.String(key),
			UploadId:   uploadID,
			PartNumber: aws.Int32(partNum),
			Body:       bytes.NewReader(buf),
		})
		if err != nil {
			return abort(err)
		}
		etag := ""
		if up.ETag != nil {
			etag = *up.ETag
		}
		completed = append(completed, types.CompletedPart{
			PartNumber: aws.Int32(partNum),
			ETag:       aws.String(etag),
		})
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
			return abort(readErr)
		}
	}
	_, err = b.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(b.cfg.Bucket),
		Key:             aws.String(key),
		UploadId:        uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	return err
}

func (b *awsBackend) DownloadMultipart(ctx context.Context, key string, w io.Writer, size int64, mp MultipartConfig) error {
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

func (b *awsBackend) PublicURL(key string) string {
	return PublicURLFromHost(b.cfg.CDNHost, key)
}

func (b *awsBackend) Close() error { return nil }

func mapAWSErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "NotFound") || strings.Contains(msg, "NoSuchKey") {
		return ErrNotFound
	}
	return err
}
