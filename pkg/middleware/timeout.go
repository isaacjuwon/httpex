package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	httperr "github.com/isaacjuwon/httpex/pkg/errors"
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

// Timeout returns a [core.Middleware] that cancels the request context after d.
//
// # Cooperative Cancellation
//
// This middleware replaces the per-request context with a timed-out version
// and checks ctx.Err() after the handler returns. For the timeout to be
// enforced, handlers and downstream middleware must respect ctx cancellation
// (e.g. pass ctx to database calls, HTTP clients, etc.).
//
// A goroutine-per-request approach is intentionally avoided here because it
// creates a data race with the pooled [core.Context]: the pool can recycle the
// context before the goroutine finishes reading from it.
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

		err := next.ServeHTTPX(c)

		// If the context deadline was exceeded, return a timeout error regardless
		// of what the handler returned (it may have returned nil or a partial error).
		if ctx.Err() != nil {
			return httperr.NewHTTPError(m.cfg.code, m.cfg.message)
		}
		return err
	})
}
