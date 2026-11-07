# Changelog

## [Unreleased]

## [0.2.0] - 2026-11-07

### Breaking

- See repository diff for API changes in this release (secure defaults, renamed packages where applicable, context-aware constructors).

### Security

- Hardened defaults: TLS, timeouts, body limits, PII redaction where logging applies.

### Fixed

- Bug fixes from the v0.2.0 hardening pass.

### Changed

- CI uses go-version-file; added race, staticcheck, govulncheck, golangci-lint (gosec).



- **BREAKING:** `NewStorage(ctx, cfg)` now takes a context bounding the startup ping/retry.
- `GetRange` validates offset/length and rejects `offset+length` overflow (`ErrInvalidRange`).
- Multipart abort failures are joined into the returned error; abort uses a non-canceled context.
- `osscred`: `json.Marshal` error is returned; `coscred`: `GetCredential` respects `ctx`.
- Retry helper uses `time.NewTimer`, `math/rand/v2`, and honors `ctx` while backing off.

## [0.1.0] - 2026-11-07

### Added

- Unified `Storage` API for Aliyun OSS, Tencent COS, WeChat COS, AWS S3, and Azure Blob
- Multipart upload and download with configurable thresholds
- `coscred` and `osscred` subpackages for temporary client upload credentials
- English README, examples, and package documentation
