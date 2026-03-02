package middleware

import (
	"net/http"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// ----- Body Limit Middleware -----

type bodyLimitMiddleware struct {
	maxBytes int64
}

// BodyLimit returns a [core.Middleware] that limits the request body size.
func BodyLimit(maxBytes int64) core.Middleware {
	return &bodyLimitMiddleware{maxBytes: maxBytes}
}

func (m *bodyLimitMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		if c.Request().Body != nil {
			c.Request().Body = http.MaxBytesReader(c.ResponseWriter(), c.Request().Body, m.maxBytes)
		}
		return next.ServeHTTPX(c)
	})
}
