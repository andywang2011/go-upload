package upload

import (
	"bytes"
	"context"
	"testing"
)

func TestHasPrefix(t *testing.T) {
	if !hasPrefix("abc", "ab") || hasPrefix("a", "ab") {
		t.Fatal()
	}
}

func TestMemBackend_getRangeEOF(t *testing.T) {
	mb := newMemBackend("https://cdn")
	mb.objects["k"] = []byte("ab")
	err := mb.GetRange(context.Background(), "k", &bytes.Buffer{}, 5, 1)
	if err == nil {
		t.Fatal("expected EOF")
	}
}

func TestMemBackend_listTruncated(t *testing.T) {
	mb := newMemBackend("https://cdn")
	mb.objects["a"] = []byte("1")
	mb.objects["b"] = []byte("2")
	_, page, err := mb.List(context.Background(), "", listOptions{maxKeys: 1})
	if err != nil || page == nil || !page.IsTruncated {
		t.Fatal(err, page)
	}
}
