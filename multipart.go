package upload

import (
	"context"
	"io"
)

func uploadInParts(ctx context.Context, b objectBackend, key string, r io.Reader, size int64, mp MultipartConfig) error {
	if size <= 0 {
		return b.UploadMultipart(ctx, key, r, size, mp)
	}
	partSize := mp.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	if size <= partSize {
		return b.Put(ctx, key, r, size, "")
	}
	return b.UploadMultipart(ctx, key, r, size, mp)
}

func downloadInParts(ctx context.Context, b objectBackend, key string, w io.Writer, size int64, mp MultipartConfig) error {
	if size <= 0 {
		size, err := b.Head(ctx, key)
		if err != nil {
			return err
		}
		if size <= 0 {
			return b.Get(ctx, key, w)
		}
	}
	partSize := mp.DownloadPartSize
	if partSize <= 0 {
		partSize = mp.PartSize
	}
	if size <= partSize {
		return b.Get(ctx, key, w)
	}
	return b.DownloadMultipart(ctx, key, w, size, mp)
}

// sequentialPartReader reads fixed-size chunks from r for multipart upload helpers.
type sequentialPartReader struct {
	r        io.Reader
	remain   int64
	partSize int64
}

func (s *sequentialPartReader) next() ([]byte, error) {
	if s.remain <= 0 {
		return nil, io.EOF
	}
	n := s.partSize
	if n > s.remain {
		n = s.remain
	}
	buf := make([]byte, n)
	got, err := io.ReadFull(s.r, buf)
	if got > 0 {
		s.remain -= int64(got)
		return buf[:got], nil
	}
	return nil, err
}

