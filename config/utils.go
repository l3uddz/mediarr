package config

import "fmt"

func GetProviderSetting(providerCfg map[string]string, key string) (*string, error) {
	v, exists := providerCfg[key]
	if !exists {
		return nil, fmt.Errorf("no provider setting found for: %q", key)
	}

	return &v, nil
}
