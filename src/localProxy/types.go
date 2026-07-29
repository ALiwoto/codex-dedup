package localProxy

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

type LocalProxyOptions struct {
	BindAddress string
	ProviderURL *url.URL
}

type LocalProxy struct {
	app               *fiber.App
	bindAddress       string
	providerHost      string
	providerURLPrefix string
	providerClient    *fasthttp.Client
}

type mutableHTTPHeader interface {
	Peek(key string) []byte
	Del(key string)
}
