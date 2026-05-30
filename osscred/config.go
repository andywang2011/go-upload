package osscred

import (
	"errors"
)

const defaultSessionName = "upload"

// Config holds Aliyun STS settings for OSS post policy uploads.
type Config struct {
	AccessKeyID     string
	AccessKeySecret string
	RoleARN         string
	SessionName     string
}

// DefaultConfig returns OSS STS defaults.
func DefaultConfig() Config {
	return Config{SessionName: defaultSessionName}
}

// Validate checks required fields and applies defaults.
func (c *Config) Validate() error {
	if c.AccessKeyID == "" || c.AccessKeySecret == "" || c.RoleARN == "" {
		return errors.New("osscred: access_key_id, access_key_secret, and role_arn are required")
	}
	if c.SessionName == "" {
		c.SessionName = defaultSessionName
	}
	return nil
}
