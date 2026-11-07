// Example: configure Aliyun OSS and upload a small object.
// Set OSS_* env vars or edit the config below. Not run in CI.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/andywang2011/go-upload"
)

func main() {
	cfg := upload.ConfigDefault()
	cfg.Provider.Name = upload.ProviderAliyun
	cfg.Aliyun = upload.AliyunConfig{
		Endpoint:        env("OSS_ENDPOINT", "oss-cn-hangzhou.aliyuncs.com"),
		AccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
		AccessKeySecret: os.Getenv("OSS_ACCESS_KEY_SECRET"),
		Bucket:          env("OSS_BUCKET", "my-bucket"),
		CDNHost:         os.Getenv("OSS_CDN_HOST"),
	}
	if cfg.Aliyun.AccessKeyID == "" {
		fmt.Println("set OSS_ACCESS_KEY_ID / OSS_ACCESS_KEY_SECRET")
		os.Exit(1)
	}
	ctx := context.Background()
	store, err := upload.NewStorage(ctx, cfg)
	if err != nil {
		fmt.Println("new storage:", err)
		os.Exit(1)
	}
	defer store.Close()

	body := strings.NewReader("hello from go-upload example")
	key := "examples/hello.txt"
	if err := store.Upload(ctx, key, body, upload.WithContentLength(int64(len("hello from go-upload example")))); err != nil {
		fmt.Println("upload:", err)
		os.Exit(1)
	}
	fmt.Println("uploaded:", store.PublicURL(key))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
