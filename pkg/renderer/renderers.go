package renderer

import (
	"html/template"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// JSONRenderer is the default Core Renderer.
type JSONRenderer struct {
	Indent bool
}

// Render writes JSON encoding of data to the response.
func (j *JSONRenderer) Render(c core.Context, code int, data any) error {
	return c.JSON(code, data)
}

// HTMLRenderer is a Renderer that executes Go html/templates.
type HTMLRenderer struct {
	Templates *template.Template
}

// Render executes the primary template.
func (h *HTMLRenderer) Render(c core.Context, code int, data any) error {
	c.ResponseWriter().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter().WriteHeader(code)
	return h.Templates.Execute(c.ResponseWriter(), data)
}

// RenderName executes a specific named template.
func (h *HTMLRenderer) RenderName(c core.Context, code int, name string, data any) error {
	c.ResponseWriter().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.ResponseWriter().WriteHeader(code)
	return h.Templates.ExecuteTemplate(c.ResponseWriter(), name, data)
}

// HTML helper to ensure type safety if using HTMLRenderer
func IsHTML(r core.Renderer) (*HTMLRenderer, bool) {
	hr, ok := r.(*HTMLRenderer)
	return hr, ok
}
