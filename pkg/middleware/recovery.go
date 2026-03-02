package middleware

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/isaacjuwon/httpex/pkg/core"
	httperr "github.com/isaacjuwon/httpex/pkg/errors"
	"github.com/isaacjuwon/httpex/pkg/logger"
)

// ----- Recovery Middleware -----

type recoveryConfig struct {
	logger  core.Logger
	handler func(c core.Context, err any)
}

// RecoveryOption configures the [Recovery] middleware.
type RecoveryOption func(*recoveryConfig)

// WithRecoveryLogger sets the logger for panic stack traces.
func WithRecoveryLogger(l core.Logger) RecoveryOption {
	return func(c *recoveryConfig) { c.logger = l }
}

// WithRecoveryHandler sets a custom function to handle panics.
func WithRecoveryHandler(fn func(c core.Context, err any)) RecoveryOption {
	return func(c *recoveryConfig) { c.handler = fn }
}

type recoveryMiddleware struct {
	cfg recoveryConfig
}

// Recovery returns a [core.Middleware] that catches panics.
func Recovery(opts ...RecoveryOption) core.Middleware {
	cfg := recoveryConfig{
		logger: logger.NewDefaultLogger(),
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &recoveryMiddleware{cfg: cfg}
}

func (m *recoveryMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				if m.cfg.handler != nil {
					m.cfg.handler(c, r)
					return
				}

				// Capture stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])

				m.cfg.logger.Error("panic recovered",
					"error", fmt.Sprintf("%v", r),
					"stack", stack,
					"method", c.Method(),
					"path", c.Path(),
				)

				returnErr = httperr.NewHTTPError(
					http.StatusInternalServerError,
					"Internal Server Error",
				)
			}
		}()

		return next.ServeHTTPX(c)
	})
}
