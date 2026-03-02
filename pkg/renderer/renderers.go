package renderer

import (
	"bytes"
	"encoding/json"
	"html/template"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// JSONRenderer is the default Core Renderer.
type JSONRenderer struct {
	// Indent enables pretty-printed JSON output when true.
	Indent bool
}

// Render writes JSON encoding of data to the response.
func (j *JSONRenderer) Render(c core.Context, code int, data any) error {
	w := c.ResponseWriter()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	if j.Indent {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(data)
}

// HTMLRenderer is a Renderer that executes Go html/templates.
type HTMLRenderer struct {
	Templates *template.Template
}

// Render executes the primary template, buffering output before writing the
// status code so that template errors can still result in a proper 500
// rather than a partially-written response.
func (h *HTMLRenderer) Render(c core.Context, code int, data any) error {
	return h.renderTo(c, code, func(buf *bytes.Buffer) error {
		return h.Templates.Execute(buf, data)
	})
}

// RenderName executes a specific named template, buffering before writing.
func (h *HTMLRenderer) RenderName(c core.Context, code int, name string, data any) error {
	return h.renderTo(c, code, func(buf *bytes.Buffer) error {
		return h.Templates.ExecuteTemplate(buf, name, data)
	})
}

func (h *HTMLRenderer) renderTo(c core.Context, code int, fn func(*bytes.Buffer) error) error {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		// Template failed before any bytes were sent — the error handler can
		// still write a proper 500 response.
		return err
	}
	w := c.ResponseWriter()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, err := buf.WriteTo(w)
	return err
}

// IsHTML reports whether r is an *HTMLRenderer and returns it.
// Helper to ensure type safety if using [HTMLRenderer].
func IsHTML(r core.Renderer) (*HTMLRenderer, bool) {
	hr, ok := r.(*HTMLRenderer)
	return hr, ok
}
