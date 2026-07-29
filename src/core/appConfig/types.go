package appConfig

import "net/url"

type PlatformConfig struct {
	Debug        bool   `section:"main" key:"debug" default:"false"`
	LogDirectory string `section:"main" key:"log_directory" default:"logs"`

	LocalBindAddress string `section:"local" key:"bind_address" default:"127.0.0.1:1918"`
	RemoteURL        string `section:"local" key:"remote_url"`
	ProviderURL      string `section:"local" key:"provider_url"`

	RemoteBindAddress string `section:"remote" key:"bind_address" default:":1919"`

	providerURL *url.URL
}
