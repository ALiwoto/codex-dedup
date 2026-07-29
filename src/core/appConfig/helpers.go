package appConfig

import (
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
	if err = validateProxyConfig(config, proxyRole); err != nil {
		return err
	}

	TheConfig = config
	return nil
}

func IsDebug() bool {
	return TheConfig.Debug
}

func GetLogDirectory() string {
	return TheConfig.LogDirectory
}
