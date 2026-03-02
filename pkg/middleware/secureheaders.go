package middleware

import (
	"fmt"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// ----- Secure Headers Middleware -----

type secureHeadersConfig struct {
	headers map[string]string
}

// SecureHeadersOption configures the [SecureHeaders] middleware.
type SecureHeadersOption func(*secureHeadersConfig)

// WithHSTS enables HTTP Strict Transport Security.
func WithHSTS(maxAge int) SecureHeadersOption {
	return func(c *secureHeadersConfig) {
		c.headers["Strict-Transport-Security"] = fmt.Sprintf("max-age=%d; includeSubDomains", maxAge)
	}
}

// WithCSP sets the Content-Security-Policy header.
func WithCSP(policy string) SecureHeadersOption {
	return func(c *secureHeadersConfig) {
		c.headers["Content-Security-Policy"] = policy
	}
}

// WithCustomHeader sets an arbitrary security header.
func WithCustomHeader(key, value string) SecureHeadersOption {
	return func(c *secureHeadersConfig) {
		c.headers[key] = value
	}
}

type secureHeadersMiddleware struct {
	cfg secureHeadersConfig
}

// SecureHeaders returns a [core.Middleware] that sets secure response headers.
func SecureHeaders(opts ...SecureHeadersOption) core.Middleware {
	cfg := secureHeadersConfig{
		headers: map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
			"X-XSS-Protection":       "0",
		},
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &secureHeadersMiddleware{cfg: cfg}
}

func (m *secureHeadersMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		for k, v := range m.cfg.headers {
			c.ResponseWriter().Header().Set(k, v)
		}
		return next.ServeHTTPX(c)
	})
}
