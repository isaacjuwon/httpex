package mux

import (
	"net/http"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// Group is a route group with a shared path prefix and its own middleware stack.
type Group struct {
	mux         *Mux
	prefix      string
	middlewares []core.Middleware
}

// Group creates a new [Group] from the Mux.
func (m *Mux) Group(prefix string) *Group {
	return &Group{mux: m, prefix: prefix}
}

// Use appends middlewares to this group.
func (g *Group) Use(mws ...core.Middleware) {
	g.middlewares = append(g.middlewares, mws...)
}

// Handle registers a handler under this group's prefix.
func (g *Group) Handle(method, path string, h core.Handler) {
	wrapped := h
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		wrapped = g.middlewares[i].Wrap(wrapped)
	}
	g.mux.router.Add(method, g.prefix+path, wrapped)
}

// Get registers a GET handler.
func (g *Group) Get(path string, h core.HandlerFunc) { g.Handle(http.MethodGet, path, h) }

// Post registers a POST handler.
func (g *Group) Post(path string, h core.HandlerFunc) { g.Handle(http.MethodPost, path, h) }

// Put registers a PUT handler.
func (g *Group) Put(path string, h core.HandlerFunc) { g.Handle(http.MethodPut, path, h) }

// Patch registers a PATCH handler.
func (g *Group) Patch(path string, h core.HandlerFunc) { g.Handle(http.MethodPatch, path, h) }

// Delete registers a DELETE handler.
func (g *Group) Delete(path string, h core.HandlerFunc) { g.Handle(http.MethodDelete, path, h) }

// Group creates a sub-group from an existing group.
func (g *Group) Group(prefix string) *Group {
	return &Group{
		mux:         g.mux,
		prefix:      g.prefix + prefix,
		middlewares: append([]core.Middleware{}, g.middlewares...),
	}
}
