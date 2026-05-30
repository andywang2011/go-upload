package osscred

import (
	"fmt"
	"time"
)

// ServiceError is returned when Aliyun STS responds with a non-200 status.
type ServiceError struct {
	Code       string `json:"Code"`
	Message    string `json:"Message"`
	RequestID  string `json:"RequestId"`
	HostID     string `json:"HostId"`
	RawMessage string
	StatusCode int
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("osscred: sts error: StatusCode=%d, ErrorCode=%s, ErrorMessage=%s, RequestId=%s",
		e.StatusCode, e.Code, e.Message, e.RequestID)
}

// Credentials holds temporary STS keys.
type Credentials struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	AccessKeySecret string    `json:"AccessKeySecret"`
	Expiration      time.Time `json:"Expiration"`
	SecurityToken   string    `json:"SecurityToken"`
}

// AssumedRoleUser is metadata from AssumeRole.
type AssumedRoleUser struct {
	Arn           string `json:"Arn"`
	AssumedRoleID string `json:"AssumedRoleId"`
}

// AssumeRoleResponse is the Aliyun STS AssumeRole JSON body.
type AssumeRoleResponse struct {
	Credentials     Credentials     `json:"Credentials"`
	AssumedRoleUser AssumedRoleUser `json:"AssumedRoleUser"`
	RequestID       string          `json:"RequestId"`
}

type postPolicyConfig struct {
	Expiration string     `json:"expiration"`
	Conditions [][]string `json:"conditions"`
}

// PolicyToken is returned to the client for OSS direct post upload.
type PolicyToken struct {
	AccessKeyID   string `json:"accessKeyId"`
	SecurityToken string `json:"securityToken"`
	Signature     string `json:"signature"`
	Policy        string `json:"policy"`
	UploadPath    string `json:"upload_path"`
}
