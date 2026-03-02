# httpex Documentation


`httpex` is a modern, lightweight HTTP toolkit for Go. It is built on a modular, interface-driven architecture that extends the standard library's `net/http` to provide grouping, path parameters, type-safe generic context extraction, fast routing, and streamlined error handling.

---

## 1. Getting Started

```bash
go get github.com/isaacjuwon/httpex
```

### Basic Server

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/isaacjuwon/httpex/pkg/core"
    "github.com/isaacjuwon/httpex/pkg/middleware"
    "github.com/isaacjuwon/httpex/pkg/mux"
    "github.com/isaacjuwon/httpex/pkg/shutdown"
)

func main() {
    m := mux.New()

    // Middlewares are executed left to right
    m.Use(
        middleware.RequestID(),
        middleware.Logging(),
        middleware.Recovery(),
    )

    m.Get("/hello/:name", func(c core.Context) error {
        return c.JSON(http.StatusOK, map[string]string{
            "message": "Hello, " + c.Param("name"),
        })
    })

    // ListenAndServe with graceful shutdown
    shutdown.ListenAndServe(
        &http.Server{Addr: ":8080", Handler: m},
        shutdown.WithTimeout(10*time.Second),
        // Callbacks now receive context so they respect the shutdown deadline
        shutdown.WithOnShutdown(func(ctx context.Context) {
            db.Close() // example cleanup
        }),
    )
}
```

---

## 2. The Context Interface

The `core.Context` interface is the heart of every request. The default implementation is pooled using `sync.Pool` to ensure **zero allocations** on the hot-path.

The per-request context is stored directly on `contextImpl` — `SetContext` does **not** allocate a new `*http.Request`.

### 2.1 Reading Data
- `c.Param("id")` — Path parameter from `/users/:id`
- `c.Query("q")` — Query parameter `?q=search` *(lazily parsed once per request)*
- `c.Header("Authorization")` — HTTP Header
- `c.RealIP()` — Safely inspects `X-Forwarded-For`, `X-Real-Ip`, and `RemoteAddr`.

### 2.2 Go Generics (Type-Safe Binding & Storage)
Instead of dealing with pointers and type-assertions, `httpex` uses generics:

```go
type CreateUserReq struct {
    Email string `json:"email"`
}

m.Post("/upload", func(c core.Context) error {
    // 1. Instantiates a CreateUserReq
    // 2. Unmarshals the JSON body into it
    // 3. Returns it ready to use!
    req, err := mux.BindValue[CreateUserReq](c)
    if err != nil {
        return err // Automatically yields HTTP 400
    }

    return c.String(http.StatusCreated, "Created: "+req.Email)
})
```

Likewise, passing values down the middleware chain is strictly typed:

```go
// In middleware:
c.Set("user_id", 42)

// In handler:
// Safely unpack it without `v.(int)` panics
userId, ok := mux.Value[int](c, "user_id")
```

### 2.3 Wiring Responses
- `c.String(http.StatusOK, "OK")`
- `c.JSON(http.StatusCreated, struct{}{})`
- `c.HTML(http.StatusOK, "index.html", data)` (requires [HTMLRenderer](#5-html-templating))
- `c.Blob(http.StatusOK, "image/png", bytes)`
- `c.NoContent(http.StatusNoContent)`
- `c.Redirect(http.StatusMovedPermanently, "https://example.com")`

---

## 3. Error Handling

Handlers in `httpex` return `error`. You don't need to manually check if headers were written or panic internally.

```go
m.Get("/fetch", func(c core.Context) error {
    data, err := db.Load()
    if err != nil {
        return err
        // Generates: {"error":"Internal Server Error"} (HTTP 500)
        // Raw error strings are NOT forwarded to the client.
    }
    return c.JSON(http.StatusOK, data)
})
```

If you want a specific status code, return an `*httperr.HTTPError` from the `pkg/errors` package:

```go
import httperr "github.com/isaacjuwon/httpex/pkg/errors"

if !found {
    return httperr.NewHTTPError(http.StatusNotFound, "User missing")
}
```

> **Note:** The package path is `pkg/errors` but the package name is `httperr`.
> Always import it with an explicit alias to avoid shadowing the stdlib `errors` package.

The `DefaultErrorHandler` uses `errors.As` internally, so wrapped `*HTTPError` values are handled correctly:

```go
// This works — the 404 is correctly extracted even when wrapped:
return fmt.Errorf("lookup failed: %w", httperr.NewHTTPError(http.StatusNotFound, "not found"))
```

### 3.1 Custom HTML Error Pages

```go
import (
    "errors"
    "github.com/isaacjuwon/httpex/pkg/core"
    "github.com/isaacjuwon/httpex/pkg/mux"
    "github.com/isaacjuwon/httpex/pkg/renderer"
    httperr "github.com/isaacjuwon/httpex/pkg/errors"
)

m := mux.New(
    mux.WithRenderer(&renderer.HTMLRenderer{Templates: tmpl}),

    mux.WithNotFound(core.HandlerFunc(func(c core.Context) error {
        return c.HTML(http.StatusNotFound, "404.html", nil)
    })),

    mux.WithErrorHandler(func(c core.Context, err error) {
        code := http.StatusInternalServerError
        var he *httperr.HTTPError
        if errors.As(err, &he) { // use errors.As, not type assertion
            code = he.Code
        }
        _ = c.HTML(code, "error.html", map[string]any{
            "Code": code,
        })
    }),
)
```

---

## 4. Routing & Grouping

The router implementation is isolated in `pkg/router` and wraps a fast radix tree.

```go
api := m.Group("/api/v1")
api.Use(middleware.CORS())

// Inherits /api/v1 prefix and CORS middleware
users := api.Group("/users")
users.Get("/", GetUsers)
users.Get("/:id", GetUserByID)
```

> **Important:** Group middleware is applied at **registration time**, not dispatch time.
> Call `Use()` on a group *before* registering routes on it, or the middleware will not apply.

---

## 5. HTML Templating

By default, `httpex` uses the `JSONRenderer`. For server-rendered HTML, inject the `HTMLRenderer`:

```go
import (
    "html/template"
    "github.com/isaacjuwon/httpex/pkg/mux"
    "github.com/isaacjuwon/httpex/pkg/renderer"
)

tmpl := template.Must(template.ParseGlob("views/*.html"))

m := mux.New(
    mux.WithRenderer(&renderer.HTMLRenderer{Templates: tmpl}),
)

m.Get("/", func(c core.Context) error {
    return c.HTML(http.StatusOK, "index.html", map[string]string{
        "Title": "Welcome!",
    })
})
```

> Template output is buffered before writing the HTTP status code.
> Template errors yield a proper 500 rather than a partial response.

To enable pretty-printed JSON from the default renderer:

```go
m := mux.New(
    mux.WithRenderer(&renderer.JSONRenderer{Indent: true}),
)
```

---

## 6. Middlewares

The `pkg/middleware` package provides essential protections.

```go
m.Use(
    middleware.RequestID(),                    // Sets X-Request-Id header
    middleware.Logging(),                      // Structured slog request logs
    middleware.Recovery(),                     // Prevents full server crashes on panic
    middleware.SecureHeaders(),                // HSTS, XSS protections, framing limits
    middleware.BodyLimit(2 * 1024 * 1024),    // Max 2MB incoming body
    middleware.Timeout(5 * time.Second),       // Cooperative context cancellation
    middleware.CORS(),                         // Flexible Cross-Origin rules
)
```

### 6.1 Logging Level

`WithLogLevel` accepts a `slog.Level` value directly:

```go
import "log/slog"

middleware.Logging(
    middleware.WithLogLevel(slog.LevelDebug),
)
```

### 6.2 Timeout — Cooperative Cancellation

The `Timeout` middleware injects a deadline into the request context and checks `ctx.Err()` after the handler returns. Handlers **must** propagate `c.Context()` to any blocking I/O for the timeout to take effect:

```go
m.Get("/slow", func(c core.Context) error {
    rows, err := db.QueryContext(c.Context(), "SELECT ...") // ← pass context
    if err != nil {
        return err
    }
    // ...
})
```

---

## 7. Custom Logger

Implement `core.Logger` to wire any structured logger:

```go
type Logger interface {
    Info(msg string, attrs ...any)
    Error(msg string, attrs ...any)
    Log(ctx context.Context, level slog.Level, msg string, attrs ...any)
}
```

A `pkg/logger.SlogAdapter` is provided out of the box:

```go
import (
    "log/slog"
    "github.com/isaacjuwon/httpex/pkg/logger"
)

l := logger.NewSlogAdapter(slog.Default())
middleware.Logging(middleware.WithLogger(l))
```

---

## 8. Configuration & Options

`httpex` is configured explicitly via functional options passed to `mux.New()`.

- `WithRenderer(core.Renderer)`
- `WithErrorHandler(httperr.ErrorHandler)`
- `WithRouter(core.Router)`
- `WithNotFound(core.Handler)`
- `WithMethodNotAllowed(core.Handler)`

```go
// Custom 404 handler
m := mux.New(
    mux.WithNotFound(core.HandlerFunc(func(c core.Context) error {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
    })),
)
```
