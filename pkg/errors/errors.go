package errors

import (
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
func DefaultErrorHandler(c core.Context, err error) {
	if c.Written() {
		return
	}

	code := http.StatusInternalServerError
	msg := err.Error()

	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
	}

	_ = c.JSON(code, map[string]string{"error": msg})
}
