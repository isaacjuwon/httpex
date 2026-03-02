package mux

import (
	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/errors"
)

// Option configures a [Mux].
type Option func(*Mux)

// WithRenderer sets a custom [core.Renderer].
func WithRenderer(r core.Renderer) Option {
	return func(m *Mux) { m.renderer = r }
}

// WithErrorHandler sets a custom [errors.ErrorHandler].
func WithErrorHandler(h errors.ErrorHandler) Option {
	return func(m *Mux) { m.errorHandler = h }
}

// WithNotFound sets the handler for unmatched routes.
func WithNotFound(h core.Handler) Option {
	return func(m *Mux) { m.notFound = h }
}

// WithMethodNotAllowed sets the handler for method-mismatched routes.
func WithMethodNotAllowed(h core.Handler) Option {
	return func(m *Mux) { m.methodNotAllowed = h }
}

// WithRouter sets a custom [core.Router] implementation.
func WithRouter(r core.Router) Option {
	return func(m *Mux) { m.router = r }
}
