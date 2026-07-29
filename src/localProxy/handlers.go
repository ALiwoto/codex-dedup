package localProxy

import (
	"errors"

	"github.com/ALiwoto/codex-dedup/src/core/utils/logging"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

func handleRequest(c fiber.Ctx) error {
	removeHopByHopHeaders(&c.Request().Header)
	c.Request().Header.SetHost(theLocalProxy.providerHost)

	err := proxy.Do(
		c,
		theLocalProxy.providerURLPrefix+c.OriginalURL(),
		theLocalProxy.providerClient,
	)
	if err != nil {
		logging.Error("provider request failed")
		return fiber.NewError(fiber.StatusBadGateway, "provider request failed")
	}

	removeHopByHopHeaders(&c.Response().Header)
	return nil
}

func handleProxyError(c fiber.Ctx, err error) error {
	statusCode := fiber.StatusInternalServerError
	message := "internal proxy error"

	if fiberError, ok := errors.AsType[*fiber.Error](err); ok {
		statusCode = fiberError.Code
		message = fiberError.Message
	}

	return c.Status(statusCode).SendString(message)
}
