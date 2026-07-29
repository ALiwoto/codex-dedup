package appStartup

import (
	"fmt"

	"github.com/ALiwoto/codex-dedup/src/core/appConfig"
	"github.com/ALiwoto/codex-dedup/src/core/utils/logging"
)

func CheckConfig(options *ConfigCheckOptions) error {
	err := appConfig.LoadConfigFromFile(options.ConfigFile, options.ProxyRole)
	if err != nil {
		return fmt.Errorf("load %s configuration: %w", options.ProxyRole, err)
	}

	closeLogger, err := logging.LoadLogger(&logging.LoggerOptions{
		Debug:        appConfig.IsDebug(),
		Verbose:      options.Verbose,
		LogDirectory: appConfig.GetLogDirectory(),
	})
	if err != nil {
		return fmt.Errorf("initialize logging: %w", err)
	}
	defer closeLogger()

	logging.Infof("%s configuration is valid", options.ProxyRole)
	logging.Debugf("debug logging enabled for %s role", options.ProxyRole)
	return nil
}
