package upload

import "errors"

var (
	ErrInvalidConfig    = errors.New("upload: invalid config")
	ErrNotInitialized   = errors.New("upload: not initialized")
	ErrNotFound         = errors.New("upload: object not found")
	ErrUnsupported      = errors.New("upload: unsupported operation")
	ErrObjectTooLarge   = errors.New("upload: object too large for simple upload")
)
