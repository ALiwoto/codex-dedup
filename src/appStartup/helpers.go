package appStartup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ALiwoto/codex-dedup/src/core/appConfig"
	"github.com/ALiwoto/codex-dedup/src/core/appValues"
	"github.com/ALiwoto/codex-dedup/src/core/utils/logging"
	"github.com/ALiwoto/codex-dedup/src/localProxy"
)

func Run(options *StartupOptions) error {
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

	logging.Debugf("debug logging enabled for %s role", options.ProxyRole)
	if options.ProxyRole != appValues.ProxyRoleLocal {
		return fmt.Errorf("remote proxy runtime is not implemented")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	proxyServer := localProxy.NewLocalProxy(&localProxy.LocalProxyOptions{
		BindAddress: appConfig.GetLocalBindAddress(),
		ProviderURL: appConfig.GetProviderURL(),
	})
	logging.Infof("local proxy listening on %s", appConfig.GetLocalBindAddress())
	return proxyServer.Run(ctx)
}
