package mux

import (
	"net/http"
	"sync"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/errors"
	"github.com/isaacjuwon/httpex/pkg/renderer"
	"github.com/isaacjuwon/httpex/pkg/router"
)

// Mux is an HTTP multiplexer that implements http.Handler.
type Mux struct {
	router           core.Router
	middlewares      []core.Middleware
	renderer         core.Renderer
	errorHandler     errors.ErrorHandler
	notFound         core.Handler
	methodNotAllowed core.Handler
	pool             sync.Pool
}

// New creates a new [Mux] with the given options.
func New(opts ...Option) *Mux {
	m := &Mux{
		renderer:     &renderer.JSONRenderer{},
		errorHandler: errors.DefaultErrorHandler,
	}

	for _, o := range opts {
		o(m)
	}

	if m.router == nil {
		m.router = router.NewRadixAdapter()
	}

	if m.notFound == nil {
		m.notFound = core.HandlerFunc(func(c core.Context) error {
			return errors.NewHTTPError(http.StatusNotFound)
		})
	}

	if m.methodNotAllowed == nil {
		m.methodNotAllowed = core.HandlerFunc(func(c core.Context) error {
			return errors.NewHTTPError(http.StatusMethodNotAllowed)
		})
	}

	m.pool = sync.Pool{
		New: func() any {
			return &contextImpl{store: make(map[string]any)}
		},
	}

	return m
}

// Handle registers a handler for the given HTTP method and path.
func (m *Mux) Handle(method, path string, h core.Handler) {
	m.router.Add(method, path, h)
}

// Get registers a GET handler.
func (m *Mux) Get(path string, h core.HandlerFunc) { m.Handle(http.MethodGet, path, h) }

// Post registers a POST handler.
func (m *Mux) Post(path string, h core.HandlerFunc) { m.Handle(http.MethodPost, path, h) }

// Put registers a PUT handler.
func (m *Mux) Put(path string, h core.HandlerFunc) { m.Handle(http.MethodPut, path, h) }

// Patch registers a PATCH handler.
func (m *Mux) Patch(path string, h core.HandlerFunc) { m.Handle(http.MethodPatch, path, h) }

// Delete registers a DELETE handler.
func (m *Mux) Delete(path string, h core.HandlerFunc) { m.Handle(http.MethodDelete, path, h) }

// Use appends middlewares to the chain.
func (m *Mux) Use(mws ...core.Middleware) {
	m.middlewares = append(m.middlewares, mws...)
}

// ServeHTTP implements http.Handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := m.pool.Get().(*contextImpl)
	c.reset(w, r, m.renderer)
	c.pool = &m.pool
	defer m.pool.Put(c)

	handler, params, found := m.router.Find(r.Method, r.URL.Path)
	if !found {
		if m.router.Has(r.URL.Path) {
			handler = m.methodNotAllowed
		} else {
			handler = m.notFound
		}
	}

	c.params = params

	// Wrap with middleware chain
	h := handler
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		h = m.middlewares[i].Wrap(h)
	}

	if err := h.ServeHTTPX(c); err != nil {
		m.errorHandler(c, err)
	}
}
