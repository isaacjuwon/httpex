package middleware

import (
	"log/slog"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/logger"
)

// ----- Logging Middleware -----

type loggingConfig struct {
	logger core.Logger
	level  slog.Level
}

// LoggingOption configures the [Logging] middleware.
type LoggingOption func(*loggingConfig)

// WithLogger sets the [core.Logger] instance.
func WithLogger(l core.Logger) LoggingOption {
	return func(c *loggingConfig) { c.logger = l }
}

// WithLogLevel sets the slog level for request logs (e.g. slog.LevelInfo).
func WithLogLevel(lvl slog.Level) LoggingOption {
	return func(c *loggingConfig) { c.level = lvl }
}

type loggingMiddleware struct {
	cfg loggingConfig
}

// Logging returns a [core.Middleware] that logs each request.
func Logging(opts ...LoggingOption) core.Middleware {
	cfg := loggingConfig{
		logger: logger.NewDefaultLogger(),
		level:  slog.LevelInfo,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &loggingMiddleware{cfg: cfg}
}

func (m *loggingMiddleware) Wrap(next core.Handler) core.Handler {
	return core.HandlerFunc(func(c core.Context) error {
		start := time.Now()

		err := next.ServeHTTPX(c)

		m.cfg.logger.Log(
			c.Context(),
			m.cfg.level,
			"request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("remote_ip", c.RealIP()),
			slog.Duration("latency", time.Since(start)),
			slog.Bool("error", err != nil),
		)

		return err
	})
}
