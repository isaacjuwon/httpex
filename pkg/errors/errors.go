// Package httperr provides HTTP-aware error types for the httpex framework.
// It is intentionally named httperr (not "errors") to avoid shadowing the
// stdlib [errors] package at import sites.
package httperr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// HTTPError represents an error that occurred during an HTTP request.
type HTTPError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

// Error implements [error].
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
	}
	return fmt.Sprintf("code=%d, message=%s", e.Code, http.StatusText(e.Code))
}

// NewHTTPError creates a new [HTTPError].
// If no message is supplied the standard HTTP status text is used.
func NewHTTPError(code int, message ...string) *HTTPError {
	he := &HTTPError{Code: code}
	if len(message) > 0 {
		he.Message = message[0]
	} else {
		he.Message = http.StatusText(code)
	}
	return he
}

// ErrorHandler is a function that handles errors returned by [Handler]s.
type ErrorHandler func(c core.Context, err error)

// DefaultErrorHandler is the default [ErrorHandler].
//
// It uses [errors.As] so that wrapped HTTPErrors are unwrapped correctly.
// Non-HTTPError values are responded to with a generic "Internal Server Error"
// message — the raw error string is intentionally NOT forwarded to the client
// to avoid leaking internal details.
func DefaultErrorHandler(c core.Context, err error) {
	if c.Written() {
		return
	}

	// Safe default: never expose raw internal error messages to clients.
	code := http.StatusInternalServerError
	msg := http.StatusText(http.StatusInternalServerError)

	// Use errors.As so wrapped *HTTPError values (e.g. fmt.Errorf("…: %w", he))
	// are still handled correctly.
	var he *HTTPError
	if errors.As(err, &he) {
		code = he.Code
		msg = he.Message
	}

	_ = c.JSON(code, map[string]string{"error": msg})
}
