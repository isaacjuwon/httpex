package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// ----- Request ID Middleware -----

type requestIDConfig struct {
	header    string
	generator func() string
}

// RequestIDOption configures the [RequestID] middleware.
type RequestIDOption func(*requestIDConfig)

// WithIDHeader sets the header name used for the request ID.
func WithIDHeader(name string) RequestIDOption {
	return func(c *requestIDConfig) { c.header = name }
}

// WithIDGenerator sets the function used to generate request IDs.
func WithIDGenerator(fn func() string) RequestIDOption {
	return func(c *requestIDConfig) { c.generator = fn }
}

type requestIDMiddleware struct {
	cfg requestIDConfig
}

// RequestID returns a [core.Middleware] that injects a unique request ID.
func RequestID(opts ...RequestIDOption) core.Middleware {
	cfg := requestIDConfig{
		header:    "X-Request-ID",
		generator: defaultIDGenerator,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &requestIDMiddleware{cfg: cfg}
}

func (m *requestIDMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		// Reuse existing ID from request header if present
		id := c.Header(m.cfg.header)
		if id == "" {
			id = m.cfg.generator()
		}

		// Add to response header
		c.ResponseWriter().Header().Set(m.cfg.header, id)

		// Add to context so handlers/loggers can retrieve it
		c.Set("request_id", id)

		return next.ServeHTTPX(c)
	})
}

func defaultIDGenerator() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
