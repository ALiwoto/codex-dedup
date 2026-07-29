package appConfig

import (
	"fmt"
	"net"
	"net/url"

	"github.com/ALiwoto/codex-dedup/src/core/appValues"
)

func validateBindAddress(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	_, port, err := net.SplitHostPort(value)
	if err != nil || port == "" {
		return fmt.Errorf("%s must be a host:port address", fieldName)
	}

	return nil
}

func validateHTTPURL(value, fieldName string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute HTTP or HTTPS URL", fieldName)
	}
	if parsed.User != nil {
		return fmt.Errorf("%s must not contain credentials", fieldName)
	}

	return nil
}
func validateProxyConfig(config *PlatformConfig, proxyRole appValues.ProxyRole) error {
	if config.LogDirectory == "" {
		return fmt.Errorf("main.log_directory is required")
	}

	if proxyRole == appValues.ProxyRoleLocal {
		if err := validateBindAddress(config.LocalBindAddress, "local.bind_address"); err != nil {
			return err
		}
		if err := validateHTTPURL(config.RemoteURL, "local.remote_url"); err != nil {
			return err
		}
		return validateHTTPURL(config.ProviderURL, "local.provider_url")
	}

	return validateBindAddress(config.RemoteBindAddress, "remote.bind_address")
}
