package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/isaacjuwon/httpex/pkg/core"
	httperr "github.com/isaacjuwon/httpex/pkg/errors"
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

	// Apply global middlewares.
	m.Use(
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		// Timeout uses cooperative cancellation: pass c.Context() to any
		// blocking I/O (DB queries, HTTP clients) so the deadline propagates.
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

		if id == "0" {
			return httperr.NewHTTPError(http.StatusNotFound, "user not found")
		}

		return c.JSON(http.StatusOK, User{ID: id, Email: "user" + id + "@example.com"})
	})

	// 3. Type-Safe JSON Binding (Generics)
	m.Post("/users", func(c core.Context) error {
		payload, err := mux.BindValue[User](c)
		if err != nil {
			// Automatically returns HTTP 400 Bad Request
			return err
		}

		payload.ID = "usr_999"

		return c.JSON(http.StatusCreated, payload)
	})

	// 4. Wrapped errors are handled correctly by the DefaultErrorHandler.
	// errors.As unwraps the chain, so you can wrap HTTPErrors safely.
	m.Get("/wrapped-error", func(c core.Context) error {
		base := httperr.NewHTTPError(http.StatusForbidden, "access denied")
		return errors.New("authorization layer: " + base.Error())
	})

	// 5. Standard errors yield a generic 500 — raw messages are NOT leaked
	// to the client by the DefaultErrorHandler.
	m.Get("/internal", func(c core.Context) error {
		return errors.New("database connection string: postgres://user:secret@host/db")
		// Client receives: {"error":"Internal Server Error"} — secret is safe.
	})

	// Setup Graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: m,
	}

	_ = shutdown.ListenAndServe(srv,
		shutdown.WithTimeout(15*time.Second),
		// WithOnShutdown callbacks now receive context.Context so they can
		// respect the shutdown deadline.
		shutdown.WithOnShutdown(func(ctx context.Context) {
			// e.g. db.Close(), cache.Flush(ctx), etc.
			_ = ctx
		}),
	)
}
