package shutdown

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/logger"
)

// Option configures [ListenAndServe].
type Option func(*config)

type config struct {
	timeout    time.Duration
	signals    []os.Signal
	onShutdown []func(context.Context)
	logger     core.Logger
}

// WithTimeout sets the graceful shutdown timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithSignals sets which OS signals trigger shutdown.
func WithSignals(sigs ...os.Signal) Option {
	return func(c *config) { c.signals = sigs }
}

// WithOnShutdown registers a callback that runs during shutdown.
// The callback receives the shutdown context (with the configured timeout)
// and all callbacks are executed concurrently. If a callback panics it does
// not prevent other callbacks or the server shutdown from proceeding.
func WithOnShutdown(fn func(context.Context)) Option {
	return func(c *config) { c.onShutdown = append(c.onShutdown, fn) }
}

// WithLogger sets the logger for shutdown messages.
func WithLogger(l core.Logger) Option {
	return func(c *config) { c.logger = l }
}

// ListenAndServe starts the HTTP server and blocks until a shutdown signal is received.
func ListenAndServe(srv *http.Server, opts ...Option) error {
	cfg := config{
		timeout: 10 * time.Second,
		signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		logger:  logger.NewDefaultLogger(),
	}
	for _, o := range opts {
		o(&cfg)
	}

	// Channel to capture server startup errors
	errCh := make(chan error, 1)

	go func() {
		cfg.logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for signal or startup error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, cfg.signals...)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		cfg.logger.Info("shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	cfg.logger.Info("shutting down server", "timeout", cfg.timeout.String())

	// Run cleanup callbacks concurrently, each protected against panics.
	// All callbacks share the same deadline context.
	var wg sync.WaitGroup
	for _, fn := range cfg.onShutdown {
		wg.Add(1)
		go func(f func(context.Context)) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					cfg.logger.Error("shutdown callback panicked", "error", r)
				}
			}()
			f(ctx)
		}(fn)
	}
	wg.Wait()

	if err := srv.Shutdown(ctx); err != nil {
		cfg.logger.Error("shutdown error", "error", err.Error())
		return err
	}

	cfg.logger.Info("server stopped gracefully")
	return nil
}
