package upload

import (
	"fmt"
	"net/url"
	"strings"
)

func cosBucketURL(cfg COSConfig) (*url.URL, error) {
	if cfg.BaseURL != "" {
		u, err := url.Parse(strings.TrimSuffix(cfg.BaseURL, "/"))
		if err != nil {
			return nil, err
		}
		return u, nil
	}
	region := cfg.regionOrDefault()
	bucket := cfg.Bucket
	if cfg.AppID != "" && !strings.Contains(bucket, "-") {
		bucket = fmt.Sprintf("%s-%s", bucket, cfg.AppID)
	}
	raw := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	return url.Parse(raw)
}
