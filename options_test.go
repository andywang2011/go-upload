package upload

import "testing"

func TestPutAndGetOptions(t *testing.T) {
	po := applyPutOptions([]PutOption{WithContentType("a"), WithContentLength(1), WithMultipart()})
	if po.contentType != "a" || po.contentLength != 1 || !po.useMultipart {
		t.Fatal(po)
	}
	go2 := applyGetOptions([]GetOption{GetWithMultipart()})
	if !go2.useMultipart {
		t.Fatal()
	}
	lo := applyListOptions([]ListOption{WithMaxKeys(5), WithContinuationToken("t"), WithDelimiter("/")})
	if lo.maxKeys != 5 || lo.continuationToken != "t" || lo.delimiter != "/" {
		t.Fatal(lo)
	}
}
