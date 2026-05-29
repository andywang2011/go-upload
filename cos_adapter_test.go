package upload

import (
	"bytes"
	"context"
	"strings"
	"testing"

	cos "github.com/tencentyun/cos-go-sdk-v5"
)

func TestCosClientAdapter_methods(t *testing.T) {
	api, err := defaultNewCOSClient(COSConfig{
		SecretID: "id", SecretKey: "sec", Bucket: "mybucket", Region: "ap-shanghai",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _, _ = api.BucketGet(ctx, &cos.BucketGetOptions{MaxKeys: 1})
	_, _ = api.ObjectPut(ctx, "key", strings.NewReader("x"), nil)
	_, _ = api.ObjectGet(ctx, "key", nil)
	_, _ = api.ObjectHead(ctx, "key", nil)
	_, _ = api.ObjectDelete(ctx, "key")
	_, _, _ = api.ObjectInitiateMultipartUpload(ctx, "key", nil)
	_, _ = api.ObjectUploadPart(ctx, "key", "up", 1, bytes.NewReader([]byte("p")), nil)
	_, _, _ = api.ObjectCompleteMultipartUpload(ctx, "key", "up", nil)
	_, _ = api.ObjectAbortMultipartUpload(ctx, "key", "up")
	_ = api.BucketBaseURL()
}
