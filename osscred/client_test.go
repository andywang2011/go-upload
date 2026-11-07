package osscred

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const assumeRoleOKBody = `{"Credentials":{"AccessKeyId":"tmpId","AccessKeySecret":"tmpKey","SecurityToken":"token","Expiration":"2026-01-01T00:00:00Z"},"AssumedRoleUser":{"Arn":"arn","AssumedRoleId":"id"},"RequestId":"req1"}`

type mockDoer struct {
	body       string
	statusCode int
	err        error
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}, nil
}

func validConfig() Config {
	return Config{AccessKeyID: "a", AccessKeySecret: "b", RoleARN: "r"}
}

func TestNewClient(t *testing.T) {
	c, err := NewClient(validConfig())
	if err != nil || c == nil || c.doer == nil {
		t.Fatalf("NewClient: err=%v c=%v", err, c)
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNewClientWithDoer(t *testing.T) {
	m := &mockDoer{statusCode: 200, body: assumeRoleOKBody}
	c, err := NewClientWithDoer(validConfig(), m)
	if err != nil || c.doer != m {
		t.Fatalf("NewClientWithDoer: err=%v c=%v", err, c)
	}
}

func TestClient_AssumeRole_NilReceiver(t *testing.T) {
	var c *Client
	if _, err := c.AssumeRole(context.Background(), 900); err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_GetPostPolicy_NilReceiver(t *testing.T) {
	var c *Client
	if _, err := c.GetPostPolicy(context.Background(), "path", 900); err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_GetPostPolicy_EmptyPath(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{})
	if _, err := c.GetPostPolicy(context.Background(), "", 900); err == nil {
		t.Fatal("expected error for empty uploadPath")
	}
}

func TestClient_GetPostPolicy_Success(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: assumeRoleOKBody})
	c.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	token, err := c.GetPostPolicy(context.Background(), "uploads/1.txt", 900)
	if err != nil {
		t.Fatalf("GetPostPolicy: %v", err)
	}
	if token.AccessKeyID != "tmpId" || token.SecurityToken != "token" || token.UploadPath != "uploads/1.txt" {
		t.Errorf("token mismatch: %+v", token)
	}
	if token.Policy == "" || token.Signature == "" {
		t.Error("Policy and Signature should be set")
	}
}

func TestClient_GetPostPolicy_MinExpire(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: assumeRoleOKBody})
	c.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	if _, err := c.GetPostPolicy(context.Background(), "p", 100); err != nil {
		t.Fatalf("GetPostPolicy with short expire: %v", err)
	}
}

func TestClient_GetPostPolicy_TrimPrefix(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: assumeRoleOKBody})
	c.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	token, err := c.GetPostPolicy(context.Background(), "/foo/bar", 900)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if token.UploadPath != "foo/bar" {
		t.Errorf("UploadPath = %q", token.UploadPath)
	}
}

func TestClient_AssumeRole_HTTPError(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 403, body: `{"Code":"Forbidden","Message":"denied"}`})
	if _, err := c.AssumeRole(context.Background(), 900); err == nil {
		t.Fatal("expected HTTP 403 error")
	}
}

func TestServiceError_Error(t *testing.T) {
	se := &ServiceError{StatusCode: 403, Code: "Forbidden", Message: "denied", RequestID: "r1"}
	if s := se.Error(); s == "" || !strings.Contains(s, "403") {
		t.Errorf("Error() = %q", s)
	}
}

func TestClient_GetPostPolicy_EmptyToken(t *testing.T) {
	body := `{"Credentials":{"AccessKeyId":"id","AccessKeySecret":"secret","SecurityToken":"","Expiration":"2026-01-01T00:00:00Z"},"AssumedRoleUser":{},"RequestId":""}`
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: body})
	if _, err := c.GetPostPolicy(context.Background(), "p", 900); err == nil {
		t.Fatal("expected error when SecurityToken is empty")
	}
}

func TestClient_AssumeRole_InvalidJSON(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: "not json"})
	if _, err := c.AssumeRole(context.Background(), 900); err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestClient_AssumeRole_DoError(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{err: errors.New("do error")})
	if _, err := c.AssumeRole(context.Background(), 900); err == nil {
		t.Fatal("expected Do error")
	}
}

func TestRandomNonce(t *testing.T) {
	n1, err := randomNonce()
	if err != nil || len(n1) == 0 {
		t.Fatalf("randomNonce: %v %q", err, n1)
	}
	n2, _ := randomNonce()
	if n1 == n2 {
		t.Error("expected different nonces")
	}
}

func TestClient_GetPostPolicy_MarshalError(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: assumeRoleOKBody})
	boom := errors.New("marshal boom")
	c.marshal = func(any) ([]byte, error) { return nil, boom }
	_, err := c.GetPostPolicy(context.Background(), "uploads/1.txt", 900)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapped marshal error", err)
	}
}

func TestClient_GetPostPolicy_NilMarshalFallsBack(t *testing.T) {
	c, _ := NewClientWithDoer(validConfig(), &mockDoer{statusCode: 200, body: assumeRoleOKBody})
	c.marshal = nil
	if _, err := c.GetPostPolicy(context.Background(), "uploads/1.txt", 900); err != nil {
		t.Fatal(err)
	}
}
