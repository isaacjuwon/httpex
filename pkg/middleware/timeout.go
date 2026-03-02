package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/errors"
)

// ----- Timeout Middleware -----

type timeoutConfig struct {
	message string
	code    int
}

// TimeoutOption configures the [Timeout] middleware.
type TimeoutOption func(*timeoutConfig)

// WithTimeoutMessage sets the response body sent when the request times out.
func WithTimeoutMessage(msg string) TimeoutOption {
	return func(c *timeoutConfig) { c.message = msg }
}

// WithTimeoutCode sets the HTTP status code on timeout. Default is 504.
func WithTimeoutCode(code int) TimeoutOption {
	return func(c *timeoutConfig) { c.code = code }
}

type timeoutMiddleware struct {
	duration time.Duration
	cfg      timeoutConfig
}

// Timeout returns a [core.Middleware] that cancels the request context.
func Timeout(d time.Duration, opts ...TimeoutOption) core.Middleware {
	cfg := timeoutConfig{
		message: "Request Timeout",
		code:    http.StatusGatewayTimeout,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &timeoutMiddleware{duration: d, cfg: cfg}
}

func (m *timeoutMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		ctx, cancel := context.WithTimeout(c.Context(), m.duration)
		defer cancel()

		c.SetContext(ctx)

		done := make(chan error, 1)
		go func() {
			done <- next.ServeHTTPX(c)
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return errors.NewHTTPError(m.cfg.code, m.cfg.message)
		}
	})
}
