package osscred

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	stsSignVersion     = "1.0"
	stsAPIVersion      = "2015-04-01"
	stsHost            = "https://sts.aliyuncs.com/"
	timeFormat         = "2006-01-02T15:04:05Z"
	respBodyFormat     = "JSON"
	percentEncode      = "%2F"
	httpGet            = "GET"
	defaultHTTPTimeout = 6 * time.Second
	minExpireSeconds   = int64(900)
)

type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpDoer struct {
	client *http.Client
}

func (d *httpDoer) Do(req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

// Client issues Aliyun STS credentials and OSS post policies.
type Client struct {
	cfg  Config
	doer doer
	now  func() time.Time
	// marshal encodes the post policy; defaults to json.Marshal (replaceable in tests).
	marshal func(v any) ([]byte, error)
}

// NewClient creates an OSS STS client.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Client{
		cfg:     cfg,
		doer:    &httpDoer{client: &http.Client{Timeout: defaultHTTPTimeout}},
		now:     time.Now,
		marshal: json.Marshal,
	}, nil
}

// NewClientWithDoer injects an HTTP doer for tests.
func NewClientWithDoer(cfg Config, d doer) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("osscred: doer is nil")
	}
	return &Client{cfg: cfg, doer: d, now: time.Now, marshal: json.Marshal}, nil
}

// AssumeRole calls Aliyun STS AssumeRole.
func (c *Client) AssumeRole(ctx context.Context, expireSec int64) (*AssumeRoleResponse, error) {
	if c == nil || c.doer == nil {
		return nil, errors.New("osscred: client is not initialized")
	}
	reqURL, err := c.generateSignedURL(expireSec)
	if err != nil {
		return nil, err
	}
	body, status, err := c.sendRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}
	return c.handleResponse(body, status)
}

// GetPostPolicy returns an OSS post policy for mini program / browser direct upload.
func (c *Client) GetPostPolicy(ctx context.Context, uploadPath string, expireSec int64) (*PolicyToken, error) {
	if c == nil || c.doer == nil {
		return nil, errors.New("osscred: client is not initialized")
	}
	if uploadPath == "" {
		return nil, errors.New("osscred: uploadPath is required")
	}
	if expireSec < minExpireSeconds {
		expireSec = minExpireSeconds
	}
	uploadPath = strings.TrimPrefix(uploadPath, "/")

	response, err := c.AssumeRole(ctx, expireSec)
	if err != nil {
		return nil, err
	}
	if response.Credentials.SecurityToken == "" {
		return nil, errors.New("osscred: empty security token")
	}
	expireEnd := c.now().Unix() + expireSec
	config := postPolicyConfig{
		Expiration: expireEndUTC(expireEnd),
		Conditions: [][]string{{"eq", "$key", uploadPath}},
	}
	marshal := c.marshal
	if marshal == nil {
		marshal = json.Marshal
	}
	result, err := marshal(config)
	if err != nil {
		return nil, fmt.Errorf("osscred: marshal post policy: %w", err)
	}
	policy := base64.StdEncoding.EncodeToString(result)
	h := hmac.New(sha1.New, []byte(response.Credentials.AccessKeySecret))
	h.Write([]byte(policy))
	signedStr := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return &PolicyToken{
		AccessKeyID:   response.Credentials.AccessKeyID,
		SecurityToken: response.Credentials.SecurityToken,
		Signature:     signedStr,
		Policy:        policy,
		UploadPath:    uploadPath,
	}, nil
}

func (c *Client) generateSignedURL(expireSec int64) (string, error) {
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}
	queryStr := "SignatureVersion=" + stsSignVersion
	queryStr += "&Format=" + respBodyFormat
	queryStr += "&Timestamp=" + url.QueryEscape(c.now().UTC().Format(timeFormat))
	queryStr += "&RoleArn=" + url.QueryEscape(c.cfg.RoleARN)
	queryStr += "&RoleSessionName=" + c.cfg.SessionName
	queryStr += "&AccessKeyId=" + c.cfg.AccessKeyID
	queryStr += "&SignatureMethod=HMAC-SHA1"
	queryStr += "&Version=" + stsAPIVersion
	queryStr += "&Action=AssumeRole"
	queryStr += "&SignatureNonce=" + nonce
	queryStr += "&DurationSeconds=" + strconv.FormatInt(expireSec, 10)

	queryParams, err := url.ParseQuery(queryStr)
	if err != nil {
		return "", err
	}
	result := queryParams.Encode()
	strToSign := httpGet + "&" + percentEncode + "&" + url.QueryEscape(result)
	hashSign := hmac.New(sha1.New, []byte(c.cfg.AccessKeySecret+"&"))
	hashSign.Write([]byte(strToSign))
	signature := base64.StdEncoding.EncodeToString(hashSign.Sum(nil))
	return stsHost + "?" + queryStr + "&Signature=" + url.QueryEscape(signature), nil
}

func (c *Client) sendRequest(ctx context.Context, reqURL string) ([]byte, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, strings.NewReader(""))
	if err != nil {
		return nil, -1, err
	}
	resp, err := c.doer.Do(httpReq)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (c *Client) handleResponse(responseBody []byte, statusCode int) (*AssumeRoleResponse, error) {
	if statusCode != http.StatusOK {
		se := ServiceError{StatusCode: statusCode, RawMessage: string(responseBody)}
		_ = json.Unmarshal(responseBody, &se)
		return nil, &se
	}
	var resp AssumeRoleResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func expireEndUTC(expireEnd int64) string {
	return time.Unix(expireEnd, 0).UTC().Format(timeFormat)
}

func randomNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
