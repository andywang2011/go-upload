package coscred

import (
	"errors"
	"fmt"
)

const (
	defaultRegion          = "ap-shanghai"
	defaultDurationSeconds = 3600
)

// Config holds COS STS settings.
type Config struct {
	AppID           string
	SecretID        string
	SecretKey       string
	Bucket          string
	Region          string
	DurationSeconds int64
}

// DefaultConfig returns COS STS defaults.
func DefaultConfig() Config {
	return Config{
		Region:          defaultRegion,
		DurationSeconds: defaultDurationSeconds,
	}
}

// Validate checks required fields and applies defaults.
func (c *Config) Validate() error {
	if c.AppID == "" || c.SecretID == "" || c.SecretKey == "" || c.Bucket == "" {
		return errors.New("coscred: app_id, secret_id, secret_key, and bucket are required")
	}
	if c.Region == "" {
		c.Region = defaultRegion
	}
	if c.DurationSeconds <= 0 {
		c.DurationSeconds = defaultDurationSeconds
	}
	return nil
}

func (c Config) String() string {
	return fmt.Sprintf("coscred.Config{AppID:%q Bucket:%q Region:%q}", c.AppID, c.Bucket, c.Region)
}
