package coscred

import (
	"context"
	"errors"
	"testing"

	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

type mockGetter struct {
	result *sts.CredentialResult
	err    error
}

func (m *mockGetter) GetCredential(opt *sts.CredentialOptions) (*sts.CredentialResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func validConfig() Config {
	return Config{
		AppID: "123", SecretID: "s", SecretKey: "k", Bucket: "mybucket", Region: "ap-guangzhou",
	}
}

func TestNewClient(t *testing.T) {
	c, err := NewClient(validConfig())
	if err != nil || c == nil || c.getter == nil {
		t.Fatalf("NewClient: err=%v c=%v", err, c)
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNewClientWithGetter(t *testing.T) {
	m := &mockGetter{err: errors.New("mock err")}
	c, err := NewClientWithGetter(validConfig(), m)
	if err != nil || c.getter != m {
		t.Fatalf("NewClientWithGetter: err=%v c=%v", err, c)
	}
}

func TestClient_GetCredential_NilReceiver(t *testing.T) {
	var c *Client
	if _, err := c.GetCredential(context.Background(), "path"); err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_GetCredential_EmptyPath(t *testing.T) {
	c, _ := NewClientWithGetter(validConfig(), &mockGetter{})
	if _, err := c.GetCredential(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty uploadPath")
	}
}

func TestClient_GetCredential_GetterError(t *testing.T) {
	c, _ := NewClientWithGetter(validConfig(), &mockGetter{err: errors.New("api error")})
	if _, err := c.GetCredential(context.Background(), "uploads/1"); err == nil {
		t.Fatal("expected getter error")
	}
}

func TestClient_GetCredential_Success(t *testing.T) {
	want := &sts.CredentialResult{
		Credentials: &sts.Credentials{
			TmpSecretID:  "tmpId",
			TmpSecretKey: "tmpKey",
			SessionToken: "token",
		},
	}
	c, _ := NewClientWithGetter(validConfig(), &mockGetter{result: want})
	res, err := c.GetCredential(context.Background(), "uploads/uid/1/*")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if res.Credentials.TmpSecretID != "tmpId" {
		t.Errorf("TmpSecretID = %q", res.Credentials.TmpSecretID)
	}
}

func TestClient_GetCredential_TrimPrefix(t *testing.T) {
	c, _ := NewClientWithGetter(validConfig(), &mockGetter{
		result: &sts.CredentialResult{Credentials: &sts.Credentials{}},
	})
	if _, err := c.GetCredential(context.Background(), "/foo/bar"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}
