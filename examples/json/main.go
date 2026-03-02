package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	httperrors "github.com/isaacjuwon/httpex/pkg/errors"
	"github.com/isaacjuwon/httpex/pkg/middleware"
	"github.com/isaacjuwon/httpex/pkg/mux"
	"github.com/isaacjuwon/httpex/pkg/shutdown"
)

// User represents a simple data model used for binding and rendering.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func main() {
	// Initialize a new Mux.
	m := mux.New()

	// Apply global robust middlewares
	m.Use(
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.Timeout(5*time.Second),
	)

	// 1. Basic Health Check
	m.Get("/health", func(c core.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "up",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 2. Path Parameters
	m.Get("/users/:id", func(c core.Context) error {
		id := c.Param("id")

		// Simulate a database lookup failure
		if id == "0" {
			return httperrors.NewHTTPError(http.StatusNotFound, "user not found")
		}

		return c.JSON(http.StatusOK, User{ID: id, Email: "user" + id + "@example.com"})
	})

	// 3. Type-Safe JSON Binding (Generics)
	m.Post("/users", func(c core.Context) error {
		// Bind directly to the instantiated structural type:
		payload, err := mux.BindValue[User](c)
		if err != nil {
			// Automatically returns HTTP 400 Bad Request
			return err
		}

		// Simulate processing and persistence
		payload.ID = "usr_999"

		return c.JSON(http.StatusCreated, payload)
	})

	// 4. Default Error Handler execution
	m.Get("/panic", func(c core.Context) error {
		// Just return a standard error. The default ErrorHandler logs it
		// and outputs an HTTP 500 JSON response to the client.
		return errors.New("database connection randomly severed")
	})

	// Setup Graceful shutdown for the server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: m,
	}

	// This blocks until an interrupt signal is received, draining
	// active requests for up to 15 seconds.
	_ = shutdown.ListenAndServe(srv, shutdown.WithTimeout(15*time.Second))
}
