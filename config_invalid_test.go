package upload

import "testing"

func TestConfig_Validate_providerFields(t *testing.T) {
	cases := []struct {
		provider string
		cfg      Config
	}{
		{ProviderAliyun, func() Config { c := validTestConfig(ProviderAliyun); c.Aliyun.Endpoint = ""; return c }()},
		{ProviderTencent, func() Config { c := validTestConfig(ProviderTencent); c.Tencent.SecretID = ""; return c }()},
		{ProviderWechat, func() Config { c := validTestConfig(ProviderWechat); c.Wechat.Bucket = ""; return c }()},
		{ProviderAWS, func() Config { c := validTestConfig(ProviderAWS); c.AWS.Bucket = ""; return c }()},
		{ProviderAzure, func() Config { c := validTestConfig(ProviderAzure); c.Azure.Container = ""; return c }()},
	}
	for _, tc := range cases {
		if err := tc.cfg.Validate(); err == nil {
			t.Fatalf("%s expected error", tc.provider)
		}
	}
}
