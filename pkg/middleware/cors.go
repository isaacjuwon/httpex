package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// ----- CORS Middleware -----

type corsConfig struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           int
}

// CORSOption configures the [CORS] middleware.
type CORSOption func(*corsConfig)

// WithOrigins sets the allowed origins. Use "*" for any.
func WithOrigins(origins ...string) CORSOption {
	return func(c *corsConfig) { c.allowOrigins = origins }
}

// WithMethods sets the allowed HTTP methods.
func WithMethods(methods ...string) CORSOption {
	return func(c *corsConfig) { c.allowMethods = methods }
}

// WithHeaders sets the allowed request headers.
func WithHeaders(headers ...string) CORSOption {
	return func(c *corsConfig) { c.allowHeaders = headers }
}

// WithExposeHeaders sets headers exposed to the browser.
func WithExposeHeaders(headers ...string) CORSOption {
	return func(c *corsConfig) { c.exposeHeaders = headers }
}

// WithCredentials enables Access-Control-Allow-Credentials.
func WithCredentials(allow bool) CORSOption {
	return func(c *corsConfig) { c.allowCredentials = allow }
}

// WithMaxAge sets the preflight cache duration in seconds.
func WithMaxAge(seconds int) CORSOption {
	return func(c *corsConfig) { c.maxAge = seconds }
}

type corsMiddleware struct {
	cfg corsConfig
}

// CORS returns a [core.Middleware] that handles Cross-Origin Resource
// Sharing. It handles preflight OPTIONS requests automatically.
func CORS(opts ...CORSOption) core.Middleware {
	cfg := corsConfig{
		allowOrigins: []string{"*"},
		allowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		allowHeaders: []string{"Content-Type", "Authorization"},
		maxAge:       86400, // 24 hours
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &corsMiddleware{cfg: cfg}
}

func (m *corsMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		origin := c.Header("Origin")
		if origin == "" {
			return next.ServeHTTPX(c)
		}

		// Check if origin is allowed
		allowed := slices.Contains(m.cfg.allowOrigins, "*") || slices.Contains(m.cfg.allowOrigins, origin)
		if !allowed {
			return next.ServeHTTPX(c)
		}

		c.ResponseWriter().Header().Set("Access-Control-Allow-Origin", origin)

		if m.cfg.allowCredentials {
			c.ResponseWriter().Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if len(m.cfg.exposeHeaders) > 0 {
			c.ResponseWriter().Header().Set("Access-Control-Expose-Headers", strings.Join(m.cfg.exposeHeaders, ", "))
		}

		// Handle preflight
		if c.Method() == http.MethodOptions {
			c.ResponseWriter().Header().Set("Access-Control-Allow-Methods", strings.Join(m.cfg.allowMethods, ", "))
			c.ResponseWriter().Header().Set("Access-Control-Allow-Headers", strings.Join(m.cfg.allowHeaders, ", "))
			if m.cfg.maxAge > 0 {
				c.ResponseWriter().Header().Set("Access-Control-Max-Age", strconv.Itoa(m.cfg.maxAge))
			}
			return c.NoContent(http.StatusNoContent)
		}

		return next.ServeHTTPX(c)
	})
}
