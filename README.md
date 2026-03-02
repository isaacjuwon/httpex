# httpex

Smaller than Gin. More flexible than Echo. More Go-idiomatic than Fiber.

`httpex` is an **interface-driven HTTP toolkit** for Go. It's not a framework — it's a complement to `net/http` designed around small, composable interfaces.

## Philosophy

1. **Error returns.** Handlers return `error`. Let the framework handle the HTTP 500s.
2. **No global state.** Everything is explicit and passed via functional options.
3. **Zero-allocation hot paths.** The `Context` is pooled, and the generic radix tree router is tightly optimized.
4. **Modern Go Generics.** Type-safe context storage and request binding without pointer juggling or `any` assertions.
5. **Standard library compatibility.** The `Mux` implements `http.Handler` out of the box.

## Installation

```bash
go get github.com/isaacjuwon/httpex
```

## Quick Start

```go
package main

import (
	"errors"
	"net/http"

	"github.com/isaacjuwon/httpex/pkg/core"
	httperr "github.com/isaacjuwon/httpex/pkg/errors"
	"github.com/isaacjuwon/httpex/pkg/middleware"
	"github.com/isaacjuwon/httpex/pkg/mux"
	"github.com/isaacjuwon/httpex/pkg/shutdown"
)

func main() {
	// Create a new Mux
	m := mux.New()

	// Add middlewares (Left-to-Right execution)
	m.Use(
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
	)

	// Basic route
	m.Get("/ping", func(c core.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
	})

	// Path parameters
	m.Get("/users/:id", func(c core.Context) error {
		id := c.Param("id")
		return c.JSON(http.StatusOK, map[string]string{"user_id": id})
	})

	// Error handling
	m.Get("/error", func(c core.Context) error {
		// Standard errors yield a generic 500 — raw messages are NOT forwarded
		// to the client by the DefaultErrorHandler.
		return errors.New("database connection failed")
	})

	// Custom HTTP Errors
	m.Get("/notfound", func(c core.Context) error {
		return httperr.NewHTTPError(http.StatusNotFound, "resource missing")
	})

	// Grouping and Sub-routing
	api := m.Group("/api/v1")
	api.Use(middleware.BodyLimit(1 << 20)) // 1MB limit for this group

	api.Post("/upload", func(c core.Context) error {
		// Generic Type-Safe Binding
		type Payload struct {
			Name string `json:"name"`
		}

		payload, err := mux.BindValue[Payload](c)
		if err != nil {
			return err // Automatically yields 400 Bad Request
		}

		return c.String(http.StatusCreated, "Created: "+payload.Name)
	})

	// Graceful shutdown helper
	srv := &http.Server{Addr: ":8080", Handler: m}
	shutdown.ListenAndServe(srv)
}
```



## Built-in Middlewares

The `pkg/middleware` subpackage contains heavily audited, optional middlewares.

- **`Logging()`** — Configurable structured request logging via `slog`. Use `middleware.WithLogLevel(slog.LevelDebug)` to tune verbosity.
- **`Recovery()`** — Catches panics, logs stack traces, yields 500s.
- **`Timeout(d)`** — Injects a deadline context; handlers must propagate `c.Context()` to blocking I/O for cooperative cancellation.
- **`RequestID()`** — Injects UUIDs into context chains and response headers.
- **`BodyLimit(bytes)`** — Protection against massive payload attacks.
- **`SecureHeaders()`** — Sane XSS, framing, and MIME-sniffing defaults.
- **`CORS()`** — Comprehensive preflight caching and origin management.

## Graceful Shutdown

Handling `SIGINT` and `SIGTERM` manually is boilerplate-heavy. `pkg/shutdown` provides a clean 1-liner to drain in-flight requests safely over a given timeout.

```go
shutdown.ListenAndServe(
    &http.Server{Addr: ":8080", Handler: mux},
    shutdown.WithTimeout(15 * time.Second),
    // Callbacks receive context so they respect the shutdown deadline.
    // All callbacks run concurrently and are panic-safe.
    shutdown.WithOnShutdown(func(ctx context.Context) {
        db.Close()
    }),
)
```
