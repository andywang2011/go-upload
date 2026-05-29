# go-upload

Object storage library for Go: **upload / download / list / multipart** (large files). Package name `upload`.

```bash
go get github.com/andywang2011/go-upload@latest
```

## Providers

| Provider | Constant | Config field | Source file |
|----------|----------|--------------|-------------|
| Aliyun OSS | `aliyun` | `Aliyun` | `oss.go` |
| Tencent COS | `tencent` | `Tencent` | `cos.go` |
| WeChat (COS) | `wechat` | `Wechat` | `cos.go` (same driver) |
| Amazon AWS | `aws` | `AWS` | `aws.go` (S3 API) |
| Azure Blob | `azure` | `Azure` | `azure.go` |

## Quick start

```go
import "github.com/andywang2011/go-upload"

cfg := upload.ConfigDefault()
cfg.Provider.Name = upload.ProviderAliyun
cfg.Aliyun = upload.AliyunConfig{
    Endpoint: "oss-cn-hangzhou.aliyuncs.com",
    AccessKeyID: "...",
    AccessKeySecret: "...",
    Bucket: "my-bucket",
    CDNHost: "https://cdn.example.com",
}
store, err := upload.NewStorage(cfg)
if err != nil { /* handle */ }
defer store.Close()

_ = store.Upload(ctx, "path/file.jpg", reader, upload.WithContentLength(size))
```

## Multipart / large files

| Field | Default |
|-------|---------|
| `Multipart.PartSize` | 8 MiB |
| `Multipart.Concurrency` | 4 |
| `Multipart.Threshold` | 100 MiB |
| `Multipart.DownloadPartSize` | same as `PartSize` |

Objects at or above `Threshold` use multipart upload/download. Force with `upload.WithMultipart()` / `upload.GetWithMultipart()`.

## Legacy `Uploader` (demo-api)

```go
u := upload.AsUploader(store)
_ = u.Upload(ctx, path, reader)
url := u.CdnUrl(path)
```

## Production

- One `Storage` client per process; reuse SDK connections.
- Startup optional connectivity check with `RetryConfig` (defaults: 3 attempts, exponential backoff).
- No viper / `log.Fatal` inside the library.

## 中文摘要

五大云对象存储（阿里 OSS、腾讯 COS、微信 COS、AWS S3、Azure Blob）的统一封装：流式上传下载、列表、大文件分片。配置用 `Config` 结构体，支持 `NewStorage` 与兼容旧版的 `Uploader` 适配器。
