package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/mux"
)

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := &testLogger{buf: &buf}

	m := mux.New()
	m.Use(Logging(WithLogger(logger), WithLogLevel(core.LevelInfo)))
	m.Get("/logtest", func(c core.Context) error {
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/logtest", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "request") || !strings.Contains(logOutput, "/logtest") {
		t.Errorf("expected log output to contain request and /logtest, got %s", logOutput)
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	logger := &testLogger{buf: &buf}

	m := mux.New()
	m.Use(Recovery(WithRecoveryLogger(logger)))
	m.Get("/panic", func(c core.Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()

	// This should not crash the test suite
	m.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") || !strings.Contains(logOutput, "test panic") {
		t.Errorf("expected log output to contain panic info, got %s", logOutput)
	}
}

func TestTimeout(t *testing.T) {
	m := mux.New()
	m.Use(Timeout(50 * time.Millisecond))
	m.Get("/slow", func(c core.Context) error {
		time.Sleep(100 * time.Millisecond)
		return c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != 504 {
		t.Errorf("expected 504 Gateway Timeout, got %d", rec.Code)
	}
}

func TestRequestID(t *testing.T) {
	m := mux.New()
	m.Use(RequestID())
	m.Get("/id", func(c core.Context) error {
		id, ok := c.Get("request_id")
		if !ok {
			return c.String(500, "missing id in context")
		}
		return c.String(200, id.(string))
	})

	req := httptest.NewRequest("GET", "/id", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Errorf("expected X-Request-ID header to be set")
	}
	if rec.Body.String() != id {
		t.Errorf("expected body to match header, got %s", rec.Body.String())
	}
}

func TestSecureHeaders(t *testing.T) {
	m := mux.New()
	m.Use(SecureHeaders(WithCustomHeader("X-Custom", "test")))
	m.Get("/secure", func(c core.Context) error {
		return c.String(200, "secure")
	})

	req := httptest.NewRequest("GET", "/secure", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	h := rec.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff")
	}
	if h.Get("X-Custom") != "test" {
		t.Errorf("missing custom header")
	}
}

func TestCORS(t *testing.T) {
	m := mux.New()
	m.Use(CORS(WithOrigins("https://example.com")))
	m.Get("/cors", func(c core.Context) error {
		return c.String(200, "cors")
	})

	// Preflight
	req := httptest.NewRequest("OPTIONS", "/cors", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Errorf("expected 204 No Content for preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("missing origin header")
	}

	// Normal request
	req = httptest.NewRequest("GET", "/cors", nil)
	req.Header.Set("Origin", "https://example.com")
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("missing origin header on standard request")
	}
}

// ----- Test utilities -----

type testLogger struct {
	buf *bytes.Buffer
}

func (l *testLogger) Info(msg string, attrs ...any) {
	l.buf.WriteString(msg)
	for _, a := range attrs {
		l.buf.WriteString(fmt.Sprintf(" %v", a))
	}
	l.buf.WriteString("\n")
}

func (l *testLogger) Error(msg string, attrs ...any) {
	l.buf.WriteString(msg)
	for _, a := range attrs {
		l.buf.WriteString(fmt.Sprintf(" %v", a))
	}
	l.buf.WriteString("\n")
}

func (l *testLogger) Log(ctx context.Context, level int, msg string, attrs ...any) {
	l.buf.WriteString(msg)
	for _, a := range attrs {
		l.buf.WriteString(fmt.Sprintf(" %v", a))
	}
	l.buf.WriteString("\n")
}
