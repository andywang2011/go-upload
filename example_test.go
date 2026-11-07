package upload_test

import (
	"github.com/andywang2011/go-upload"
)

func ExampleNewStorage() {
	cfg := upload.ConfigDefault()
	cfg.Provider.Name = upload.ProviderAliyun
	cfg.Aliyun = upload.AliyunConfig{
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "...",
		AccessKeySecret: "...",
		Bucket:          "my-bucket",
	}
	store, err := upload.NewStorage(cfg)
	if err != nil {
		return
	}
	defer store.Close()
}
