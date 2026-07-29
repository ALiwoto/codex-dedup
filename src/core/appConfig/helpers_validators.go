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

func parseHTTPURL(value, fieldName string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%s must be an absolute HTTP or HTTPS URL", fieldName)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%s must not contain credentials", fieldName)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain a query or fragment", fieldName)
	}

	return parsed, nil
}

func validateHTTPURL(value, fieldName string) error {
	_, err := parseHTTPURL(value, fieldName)
	return err
}

func validateProxyConfig(config *PlatformConfig, proxyRole appValues.ProxyRole) (*url.URL, error) {
	if config.LogDirectory == "" {
		return nil, fmt.Errorf("main.log_directory is required")
	}

	if proxyRole == appValues.ProxyRoleLocal {
		if err := validateBindAddress(config.LocalBindAddress, "local.bind_address"); err != nil {
			return nil, err
		}
		if err := validateHTTPURL(config.RemoteURL, "local.remote_url"); err != nil {
			return nil, err
		}
		return parseHTTPURL(config.ProviderURL, "local.provider_url")
	}

	return nil, validateBindAddress(config.RemoteBindAddress, "remote.bind_address")
}
