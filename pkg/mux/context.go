package mux

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/errors"
	"github.com/isaacjuwon/httpex/pkg/renderer"
)

// contextImpl is the concrete implementation of core.Context, pooled via sync.Pool.
type contextImpl struct {
	req      *http.Request
	resp     http.ResponseWriter
	renderer core.Renderer
	params   core.Params
	store    map[string]any
	written  bool
	pool     *sync.Pool
}

func (c *contextImpl) reset(w http.ResponseWriter, r *http.Request, rnd core.Renderer) {
	c.req = r
	c.resp = w
	c.renderer = rnd
	c.params = c.params[:0]
	c.written = false
	for k := range c.store {
		delete(c.store, k)
	}
}

func (c *contextImpl) Param(name string) string {
	return c.params.Get(name)
}

func (c *contextImpl) SetParams(ps core.Params) {
	c.params = ps
}

func (c *contextImpl) Query(name string) string {
	return c.req.URL.Query().Get(name)
}

func (c *contextImpl) QueryDefault(name, fallback string) string {
	if v := c.Query(name); v != "" {
		return v
	}
	return fallback
}

func (c *contextImpl) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *contextImpl) Bind(v any) error {
	if c.req.Body == nil {
		return errors.NewHTTPError(http.StatusBadRequest, "missing request body")
	}
	defer c.req.Body.Close()
	if err := json.NewDecoder(c.req.Body).Decode(v); err != nil {
		return errors.NewHTTPError(http.StatusBadRequest, "invalid JSON: "+err.Error())
	}
	return nil
}

func (c *contextImpl) JSON(code int, v any) error {
	c.resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.resp.WriteHeader(code)
	c.written = true
	return json.NewEncoder(c.resp).Encode(v)
}

func (c *contextImpl) String(code int, s string) error {
	c.resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.resp.WriteHeader(code)
	c.written = true
	_, err := io.WriteString(c.resp, s)
	return err
}

func (c *contextImpl) NoContent(code int) error {
	c.resp.WriteHeader(code)
	c.written = true
	return nil
}

func (c *contextImpl) Blob(code int, contentType string, b []byte) error {
	c.resp.Header().Set("Content-Type", contentType)
	c.resp.WriteHeader(code)
	c.written = true
	_, err := c.resp.Write(b)
	return err
}

func (c *contextImpl) HTML(code int, name string, data any) error {
	if hr, ok := renderer.IsHTML(c.renderer); ok {
		return hr.RenderName(c, code, name, data)
	}
	return errors.NewHTTPError(http.StatusInternalServerError, "HTML rendering requires an HTMLRenderer")
}

func (c *contextImpl) Render(code int, data any) error {
	return c.renderer.Render(c, code, data)
}

func (c *contextImpl) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return errors.NewHTTPError(http.StatusInternalServerError, "invalid redirect code")
	}
	http.Redirect(c.resp, c.req, url, code)
	c.written = true
	return nil
}

func (c *contextImpl) Written() bool {
	return c.written
}

func (c *contextImpl) Set(key string, val any) {
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = val
}

func (c *contextImpl) Get(key string) (any, bool) {
	if c.store == nil {
		return nil, false
	}
	v, ok := c.store[key]
	return v, ok
}

func (c *contextImpl) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic("httpex: key " + key + " not found in context store")
	}
	return v
}

func (c *contextImpl) RealIP() string {
	if ip := c.req.Header.Get("X-Forwarded-For"); ip != "" {
		if i := strings.IndexByte(ip, ','); i > 0 {
			return strings.TrimSpace(ip[:i])
		}
		return ip
	}
	if ip := c.req.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	host, _, _ := net.SplitHostPort(c.req.RemoteAddr)
	return host
}

func (c *contextImpl) Path() string {
	return c.req.URL.Path
}

func (c *contextImpl) Method() string {
	return c.req.Method
}

func (c *contextImpl) Context() context.Context {
	return c.req.Context()
}

func (c *contextImpl) SetContext(ctx context.Context) {
	c.req = c.req.WithContext(ctx)
}

func (c *contextImpl) Request() *http.Request {
	return c.req
}

func (c *contextImpl) ResponseWriter() http.ResponseWriter {
	return c.resp
}

// BindValue decodes the request body as JSON and returns it instantiated.
func BindValue[T any](c core.Context) (T, error) {
	var result T
	err := c.Bind(&result)
	return result, err
}

// Value retrieves a typed value from the per-request store.
func Value[T any](c core.Context, key string) (T, bool) {
	var zero T
	v, ok := c.Get(key)
	if !ok {
		return zero, false
	}
	tVal, ok := v.(T)
	if !ok {
		return zero, false
	}
	return tVal, true
}

// MustValue retrieves a typed value and panics if not found or if type parsing fails.
func MustValue[T any](c core.Context, key string) T {
	v, ok := Value[T](c, key)
	if !ok {
		panic("httpex: key " + key + " missing or type assertion failed")
	}
	return v
}
