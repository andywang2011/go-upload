# go-upload

Object storage library for Go: upload, download, list, and multipart transfers across major cloud providers.

## Install

```bash
go get github.com/andywang2011/go-upload@v0.1.0
```

## Quick start

```go
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
	// handle
}
defer store.Close()
```

## Examples

```bash
cd go-upload/examples/aliyun && go run .
cd go-upload/examples/cos && go run .
```

Subpackages for browser STS upload: `coscred` (Tencent COS), `osscred` (Aliyun OSS).

## License

MIT — see [LICENSE](LICENSE).
