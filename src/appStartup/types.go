package appStartup

import "github.com/ALiwoto/codex-dedup/src/core/appValues"

type StartupOptions struct {
	ProxyRole  appValues.ProxyRole
	ConfigFile string
	Verbose    bool
}
