package core

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
)

// Handler processes an HTTP request through a [Context] and returns an error.
type Handler interface {
	ServeHTTPX(c Context) error
}

// HandlerFunc adapts an ordinary function into a [Handler].
type HandlerFunc func(c Context) error

// ServeHTTPX implements [Handler].
func (f HandlerFunc) ServeHTTPX(c Context) error { return f(c) }

// Middleware wraps a [Handler] with additional behavior.
type Middleware interface {
	Wrap(Handler) Handler
}

// MiddlewareFunc adapts an ordinary function into a [Middleware].
type MiddlewareFunc func(Handler) Handler

// Wrap implements [Middleware].
func (f MiddlewareFunc) Wrap(h Handler) Handler { return f(h) }

// Context is the per-request context that flows through handlers and middleware.
type Context interface {
	// Request helpers
	Param(name string) string
	Query(name string) string
	QueryDefault(name, fallback string) string
	Header(key string) string
	Bind(v any) error

	// Response helpers
	JSON(code int, v any) error
	String(code int, s string) error
	NoContent(code int) error
	Blob(code int, contentType string, b []byte) error
	HTML(code int, name string, data any) error
	Render(code int, data any) error
	Redirect(code int, url string) error
	Written() bool

	// Context store
	Set(key string, val any)
	Get(key string) (any, bool)
	MustGet(key string) any

	// Underlying stdlib access
	RealIP() string
	Path() string
	Method() string
	Context() context.Context
	SetContext(ctx context.Context)
	Request() *http.Request
	ResponseWriter() http.ResponseWriter

	// Internal/Router helpers
	SetParams(ps Params)
}

// Params holds path parameters extracted by the router.
type Params []Param

// Param is a single path parameter key-value pair.
type Param struct {
	Key   string
	Value string
}

// Get returns the value of the named parameter, or an empty string.
func (ps Params) Get(name string) string {
	if i := slices.IndexFunc(ps, func(p Param) bool { return p.Key == name }); i >= 0 {
		return ps[i].Value
	}
	return ""
}

// Router is the interface for route storage and lookup.
type Router interface {
	Add(method, path string, handler Handler)
	Find(method, path string) (handler Handler, params Params, found bool)
	Has(path string) bool
}

// Renderer writes a structured response body.
type Renderer interface {
	Render(c Context, code int, data any) error
}

// Logger is the interface for structured logging.
// attrs must be alternating key-value pairs or [slog.Attr] values,
// matching the contract of [log/slog].
type Logger interface {
	Info(msg string, attrs ...any)
	Error(msg string, attrs ...any)
	// Log logs at an arbitrary slog level.
	Log(ctx context.Context, level slog.Level, msg string, attrs ...any)
}
