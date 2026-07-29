package appConfig

import (
	"net/url"

	"github.com/ALiwoto/codex-dedup/src/core/appValues"
	"github.com/ALiwoto/ssg/ssg"
)

func LoadConfig(proxyRole appValues.ProxyRole) error {
	return LoadConfigFromFile("config.ini:virtual", proxyRole)
}

func LoadConfigFromFile(fileName string, proxyRole appValues.ProxyRole) error {
	if TheConfig != nil {
		return nil
	}

	var config = &PlatformConfig{}

	err := ssg.ParseConfig(config, fileName)
	if err != nil {
		return err
	}
	parsedProviderURL, err := validateProxyConfig(config, proxyRole)
	if err != nil {
		return err
	}

	config.providerURL = parsedProviderURL
	TheConfig = config
	return nil
}

func IsDebug() bool {
	return TheConfig.Debug
}

func GetLogDirectory() string {
	return TheConfig.LogDirectory
}

func GetLocalBindAddress() string {
	return TheConfig.LocalBindAddress
}

func GetProviderURL() *url.URL {
	result := *TheConfig.providerURL
	return &result
}
