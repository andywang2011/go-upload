package upload

import "fmt"

var buildBackend = defaultBuildBackend

func defaultBuildBackend(cfg Config) (objectBackend, error) {
	switch cfg.Provider.Name {
	case ProviderAliyun:
		return newOSSBackend(cfg.Aliyun, cfg.Multipart)
	case ProviderTencent:
		return newCOSBackend(cfg.Tencent, cfg.Multipart)
	case ProviderWechat:
		return newCOSBackend(cfg.Wechat, cfg.Multipart)
	case ProviderAWS:
		return newAWSBackend(cfg.AWS, cfg.Multipart)
	case ProviderAzure:
		return newAzureBackend(cfg.Azure, cfg.Multipart)
	default:
		return nil, fmt.Errorf("%w: provider %q", ErrInvalidConfig, cfg.Provider.Name)
	}
}
