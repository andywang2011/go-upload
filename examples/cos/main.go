// Example: Tencent / WeChat COS upload.
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
	cfg.Provider.Name = upload.ProviderTencent
	cfg.Tencent = upload.COSConfig{
		Region:    env("COS_REGION", "ap-shanghai"),
		SecretID:  os.Getenv("COS_SECRET_ID"),
		SecretKey: os.Getenv("COS_SECRET_KEY"),
		Bucket:    env("COS_BUCKET", "mybucket-1250000000"),
		CDNHost:   os.Getenv("COS_CDN_HOST"),
	}
	if cfg.Tencent.SecretID == "" {
		fmt.Println("set COS_SECRET_ID / COS_SECRET_KEY")
		os.Exit(1)
	}
	store, err := upload.NewStorage(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer store.Close()

	ctx := context.Background()
	key := "examples/cos-hello.txt"
	payload := "cos example"
	if err := store.Upload(ctx, key, strings.NewReader(payload), upload.WithContentLength(int64(len(payload)))); err != nil {
		fmt.Println("upload:", err)
		os.Exit(1)
	}
	fmt.Println("ok:", store.PublicURL(key))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
