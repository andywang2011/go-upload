package upload

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestUploadInParts_small(t *testing.T) {
	mb := newMemBackend("https://cdn")
	ctx := context.Background()
	data := []byte("tiny")
	if err := uploadInParts(ctx, mb, "k", bytes.NewReader(data), int64(len(data)), ConfigDefault().Multipart); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadInParts_large(t *testing.T) {
	mb := newMemBackend("https://cdn")
	ctx := context.Background()
	data := []byte(strings.Repeat("x", 32))
	_ = mb.Put(ctx, "big", bytes.NewReader(data), int64(len(data)), "")
	mp := MultipartConfig{PartSize: 8, DownloadPartSize: 8, Threshold: 4}
	var buf bytes.Buffer
	if err := downloadInParts(ctx, mb, "big", &buf, int64(len(data)), mp); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != len(data) {
		t.Fatalf("len=%d", buf.Len())
	}
}

func TestSequentialPartReader(t *testing.T) {
	s := &sequentialPartReader{r: bytes.NewReader([]byte("abcd")), remain: 4, partSize: 2}
	p1, err := s.next()
	if err != nil || string(p1) != "ab" {
		t.Fatal(err, p1)
	}
	p2, err := s.next()
	if err != nil || string(p2) != "cd" {
		t.Fatal(err, p2)
	}
	_, err = s.next()
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestUploadInParts_explicitLarge(t *testing.T) {
	mb := newMemBackend("https://cdn")
	ctx := context.Background()
	data := bytes.Repeat([]byte("x"), 32)
	mp := MultipartConfig{PartSize: 8, Threshold: 16}
	if err := uploadInParts(ctx, mb, "big", bytes.NewReader(data), int64(len(data)), mp); err != nil {
		t.Fatal(err)
	}
}

func TestUploadInParts_unknownSize(t *testing.T) {
	mb := newMemBackend("https://cdn")
	ctx := context.Background()
	if err := uploadInParts(ctx, mb, "k", bytes.NewReader([]byte("data")), 0, ConfigDefault().Multipart); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadInParts_headSize(t *testing.T) {
	mb := newMemBackend("https://cdn")
	ctx := context.Background()
	_ = mb.Put(ctx, "k", bytes.NewReader([]byte("12345")), 5, "")
	var buf bytes.Buffer
	mp := MultipartConfig{Threshold: 2, PartSize: 2, DownloadPartSize: 2}
	if err := downloadInParts(ctx, mb, "k", &buf, 0, mp); err != nil {
		t.Fatal(err)
	}
}

func TestMemBackend_uploadMultipart(t *testing.T) {
	mb := newMemBackend("https://cdn")
	if err := mb.UploadMultipart(context.Background(), "k", bytes.NewReader([]byte("x")), 1, ConfigDefault().Multipart); err != nil {
		t.Fatal(err)
	}
}

func TestPublicURLFromHost(t *testing.T) {
	if PublicURLFromHost("https://cdn/", "/a") != "https://cdn/a" {
		t.Fatal()
	}
	if PublicURLFromHost("https://cdn", "") != "https://cdn" {
		t.Fatal()
	}
}
