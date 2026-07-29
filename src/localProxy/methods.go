package localProxy

import (
	"context"
	"net"

	"github.com/gofiber/fiber/v3"
)

func (p *LocalProxy) Run(ctx context.Context) error {
	return p.app.Listen(p.bindAddress, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
	})
}

func (p *LocalProxy) RunWithListener(ctx context.Context, listener net.Listener) error {
	return p.app.Listener(listener, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
	})
}
