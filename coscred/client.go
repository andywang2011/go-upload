package coscred

import (
	"context"
	"fmt"
	"strings"

	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

type credentialGetter interface {
	GetCredential(opt *sts.CredentialOptions) (*sts.CredentialResult, error)
}

var _ credentialGetter = (*sts.Client)(nil)

// Client issues scoped COS temporary credentials.
type Client struct {
	cfg    Config
	getter credentialGetter
}

// NewClient creates a COS STS client.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	st := sts.NewClient(cfg.SecretID, cfg.SecretKey, nil)
	return &Client{cfg: cfg, getter: st}, nil
}

// NewClientWithGetter injects a credential getter for tests.
func NewClientWithGetter(cfg Config, getter credentialGetter) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if getter == nil {
		return nil, fmt.Errorf("coscred: getter is nil")
	}
	return &Client{cfg: cfg, getter: getter}, nil
}

// GetCredential returns temporary credentials for uploadPath (e.g. "uploads/" or "uploads/uid/1/*").
func (c *Client) GetCredential(ctx context.Context, uploadPath string) (*sts.CredentialResult, error) {
	if c == nil || c.getter == nil {
		return nil, fmt.Errorf("coscred: client is not initialized")
	}
	if uploadPath == "" {
		return nil, fmt.Errorf("coscred: uploadPath is required")
	}
	uploadPath = strings.TrimPrefix(uploadPath, "/")
	resource := fmt.Sprintf("qcs::cos:%s:uid/%s:%s/%s", c.cfg.Region, c.cfg.AppID, c.cfg.Bucket, uploadPath)

	opt := &sts.CredentialOptions{
		DurationSeconds: c.cfg.DurationSeconds,
		Region:          c.cfg.Region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					Action:   uploadPolicyActions,
					Effect:   "allow",
					Resource: []string{resource},
				},
			},
		},
	}

	// The STS SDK call is not context-aware; run it in a goroutine so the
	// caller can return as soon as ctx is done.
	type result struct {
		res *sts.CredentialResult
		err error
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ch := make(chan result, 1)
	go func() {
		res, err := c.getter.GetCredential(opt)
		ch <- result{res, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.res, r.err
	}
}

var uploadPolicyActions = []string{
	"name/cos:PostObject",
	"name/cos:PutObject",
	"name/cos:InitiateMultipartUpload",
	"name/cos:ListMultipartUploads",
	"name/cos:ListParts",
	"name/cos:UploadPart",
	"name/cos:CompleteMultipartUpload",
	"name/cos:AbortMultipartUpload",
}
