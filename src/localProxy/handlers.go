package localProxy

import (
	"bytes"
	"errors"

	"github.com/ALiwoto/codex-dedup/src/core/utils/logging"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func handleRequest(c fiber.Ctx) error {
	providerRequest := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(providerRequest)

	clientRequest := c.Request()
	clientRequest.CopyTo(providerRequest)
	removeHopByHopHeaders(&providerRequest.Header)
	providerRequest.Header.SetHost(theLocalProxy.providerHost)
	providerRequest.SetRequestURI(theLocalProxy.providerURLPrefix + c.OriginalURL())

	var measurementResults <-chan bodyMeasurementResult
	var measuredStream *measuredBodyStream
	if clientRequest.IsBodyStream() {
		measuredStream, measurementResults = startBodyMeasurement(
			clientRequest.BodyStream(),
			int64(clientRequest.Header.ContentLength()),
			theLocalProxy.measurementStore,
		)
		providerRequest.SetBodyStream(measuredStream, clientRequest.Header.ContentLength())
	} else if body := providerRequest.Body(); len(body) != 0 {
		measurement, err := measureRequestBody(
			bytes.NewReader(body),
			theLocalProxy.measurementStore,
		)
		handleBodyMeasurementResult(bodyMeasurementResult{
			measurement: measurement,
			err:         err,
		})
	}

	err := theLocalProxy.providerClient.Do(providerRequest, c.Response())
	if measuredStream != nil {
		_ = measuredStream.Close()
		handleBodyMeasurementResult(<-measurementResults)
	}
	if err != nil {
		logging.Error("provider request failed")
		return fiber.NewError(fiber.StatusBadGateway, "provider request failed")
	}

	removeHopByHopHeaders(&c.Response().Header)
	return nil
}

func handleBodyMeasurementResult(result bodyMeasurementResult) {
	if result.err != nil {
		if theLocalProxy.logMeasurements {
			logging.Error("request body measurement failed")
		}
		return
	}
	if theLocalProxy.logMeasurements {
		logBodyMeasurement(result.measurement)
	}
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
