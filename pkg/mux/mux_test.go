package mux

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/renderer"
)

func TestMuxRouting(t *testing.T) {
	m := New()

	m.Get("/users", func(c core.Context) error {
		return c.String(http.StatusOK, "get_users")
	})

	m.Post("/users", func(c core.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
	})

	m.Get("/users/:id", func(c core.Context) error {
		return c.String(http.StatusOK, c.Param("id"))
	})

	// Test GET /users
	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "get_users" {
		t.Errorf("expected get_users, got %s", rec.Body.String())
	}

	// Test GET /users/:id (Param extraction)
	req = httptest.NewRequest("GET", "/users/123", nil)
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "123" {
		t.Errorf("expected 123, got %s", rec.Body.String())
	}

	// Test 404
	req = httptest.NewRequest("GET", "/missing", nil)
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}

	// Test 405 (Method Not Allowed)
	req = httptest.NewRequest("PUT", "/users", nil)
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestMuxMiddleware(t *testing.T) {
	m := New()

	var order []string

	mw1 := core.MiddlewareFunc(func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(c core.Context) error {
			order = append(order, "mw1_before")
			err := next.ServeHTTPX(c)
			order = append(order, "mw1_after")
			return err
		})
	})

	mw2 := core.MiddlewareFunc(func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(c core.Context) error {
			order = append(order, "mw2_before")
			err := next.ServeHTTPX(c)
			order = append(order, "mw2_after")
			return err
		})
	})

	m.Use(mw1, mw2)

	m.Get("/test", func(c core.Context) error {
		order = append(order, "handler")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	expected := []string{"mw1_before", "mw2_before", "handler", "mw2_after", "mw1_after"}
	if len(order) != len(expected) {
		t.Fatalf("expected order %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, order[i])
		}
	}
}

func TestGroup(t *testing.T) {
	m := New()

	api := m.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/status", func(c core.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected ok, got %s", rec.Body.String())
	}
}

func TestContextBinding(t *testing.T) {
	m := New()
	m.Post("/echo", func(c core.Context) error {
		var payload struct {
			Message string `json:"message"`
		}
		if err := c.Bind(&payload); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, payload)
	})

	body := bytes.NewBufferString(`{"message": "hello"}`)
	req := httptest.NewRequest("POST", "/echo", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != `{"message":"hello"}`+"\n" {
		t.Errorf("expected JSON response, got %s", rec.Body.String())
	}
}

func TestContextGenericBind(t *testing.T) {
	m := New()

	type Payment struct {
		Amount int `json:"amount"`
	}

	m.Post("/pay", func(c core.Context) error {
		payment, err := BindValue[Payment](c)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, payment)
	})

	body := bytes.NewBufferString(`{"amount": 500}`)
	req := httptest.NewRequest("POST", "/pay", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != `{"amount":500}`+"\n" {
		t.Errorf("expected JSON response, got %s", rec.Body.String())
	}
}

func TestContextGenericValue(t *testing.T) {
	m := New()

	type User struct {
		ID string
	}

	m.Use(core.MiddlewareFunc(func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(c core.Context) error {
			c.Set("user", User{ID: "usr_123"})
			return next.ServeHTTPX(c)
		})
	}))

	m.Get("/me", func(c core.Context) error {
		// Type-safe retrieval without type assertion!
		user, ok := Value[User](c, "user")
		if !ok {
			return c.String(http.StatusInternalServerError, "user not found or invalid type")
		}
		return c.String(http.StatusOK, user.ID)
	})

	req := httptest.NewRequest("GET", "/me", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "usr_123" {
		t.Errorf("expected usr_123, got %s", rec.Body.String())
	}
}

func TestHTMLRenderer(t *testing.T) {
	importTmpl := `<html><body>Hello {{.Name}}!</body></html>`
	tmpl, err := template.New("index").Parse(importTmpl)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	hr := &renderer.HTMLRenderer{Templates: tmpl}
	m := New(WithRenderer(hr))

	m.Get("/html", func(c core.Context) error {
		return c.HTML(http.StatusOK, "index", struct{ Name string }{"httpex"})
	})

	req := httptest.NewRequest("GET", "/html", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != `<html><body>Hello httpex!</body></html>` {
		t.Errorf("expected HTML output, got %s", rec.Body.String())
	}
}
