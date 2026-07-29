package localProxy

import (
	"net/url"
	"strings"

	"github.com/ALiwoto/codex-dedup/src/core/appValues"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func NewLocalProxy(options *LocalProxyOptions) *LocalProxy {
	proxyServer := &LocalProxy{
		bindAddress:       options.BindAddress,
		providerHost:      options.ProviderURL.Host,
		providerURLPrefix: getProviderURLPrefix(options.ProviderURL),
		providerClient: &fasthttp.Client{
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			StreamResponseBody:            true,
		},
	}
	proxyServer.app = fiber.New(fiber.Config{
		AppName:                      appValues.ApplicationName,
		DisableDefaultContentType:    true,
		DisableDefaultDate:           true,
		DisableHeaderNormalizing:     true,
		DisablePreParseMultipartForm: true,
		ErrorHandler:                 handleProxyError,
		StreamRequestBody:            true,
	})
	theLocalProxy = proxyServer
	proxyServer.app.Use(handleRequest)

	return proxyServer
}

func getProviderURLPrefix(providerURL *url.URL) string {
	providerPath := providerURL.EscapedPath()
	if providerPath == "/" {
		providerPath = ""
	} else {
		providerPath = strings.TrimSuffix(providerPath, "/")
	}

	return providerURL.Scheme + "://" + providerURL.Host + providerPath
}

func removeHopByHopHeaders(header mutableHTTPHeader) {
	connectionHeader := string(header.Peek(fiber.HeaderConnection))
	for _, headerName := range strings.Split(connectionHeader, ",") {
		headerName = strings.TrimSpace(headerName)
		if headerName != "" {
			header.Del(headerName)
		}
	}

	for _, headerName := range hopByHopHeaderNames {
		header.Del(headerName)
	}
}
