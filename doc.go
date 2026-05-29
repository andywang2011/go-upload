// Package upload provides a thin, testable object storage client for major cloud providers.
//
// Design constraints (for use in application frameworks):
//   - No viper, no global singletons, no log.Fatal inside the library.
//   - Storage interface covers upload, download, list, delete, and multipart transfers.
//   - One Storage per process; reuse it across goroutines.
//   - Call Close on shutdown.
//
// Supported providers: Aliyun OSS, Tencent COS, WeChat (COS config), AWS S3, Azure Blob.
package upload
