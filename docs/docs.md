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
        return c.JSON(200, map[string]string{
            "message": "Hello, " + c.Param("name"),
        })
    })

    // ListenAndServe with graceful shutdown
    shutdown.ListenAndServe(&http.Server{
        Addr:    ":8080",
        Handler: m,
    })
}
```

---

## 2. The Context Interface

The `core.Context` interface is the heart of every request. The default implementation is pooled using `sync.Pool` to ensure **zero allocations** on the hot-path.

### 2.1 Reading Data
- `c.Param("id")` — Path parameter from `/users/:id`
- `c.Query("q")` — Query parameter `?q=search`
- `c.Header("Authorization")` — HTTP Header
- `c.RealIP()` — Safely inspects `X-Forwarded-For`, `X-Real-Ip`, and `RemoteAddr`.

### 2.2 Go 1.18 Generics (Type-Safe Binding & Storage)
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
    
    return c.String(200, "Created: "+req.Email)
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
- `c.String(200, "OK")`
- `c.JSON(201, struct{}{})`
- `c.HTML(200, "index.html", data)` (requires [HTMLRenderer](#5-html-templating))
- `c.Blob(200, "image/png", bytes)`
- `c.NoContent(204)`
- `c.Redirect(301, "https://google.com")`

---

## 3. Error Handling

Handlers in `httpex` return `error`. You don't need to manually check if headers were written or panic internally. 

```go
m.Get("/fetch", func(c core.Context) error {
    data, err := db.Load()
    if err != nil {
        return err 
        // Generates: {"error":"Internal Server Error"} (HTTP 500)
    }
    return c.JSON(200, data)
})
```

If you want a specific status code, return an `HTTPError` from the `errors` package:

```go
import httperrors "github.com/isaacjuwon/httpex/pkg/errors"

if !found {
    return httperrors.NewHTTPError(http.StatusNotFound, "User missing")
}
```

### 3.1 Custom HTML Error Pages

By default, the `httpex` framework responds with JSON for all errors and 404s. If you are building a server-rendered HTML application, you can easily override this to serve your own custom HTML error templates (like `404.html` and `500.html`).

You do this via the `WithErrorHandler`, `WithNotFound`, and `WithMethodNotAllowed` options in the `mux` package:

```go
import (
    "github.com/isaacjuwon/httpex/pkg/core"
    "github.com/isaacjuwon/httpex/pkg/mux"
    "github.com/isaacjuwon/httpex/pkg/renderer"
    httperrors "github.com/isaacjuwon/httpex/pkg/errors"
)

m := mux.New(
    mux.WithRenderer(&renderer.HTMLRenderer{Templates: tmpl}),
    
    // Override the default 404 handler
    mux.WithNotFound(core.HandlerFunc(func(c core.Context) error {
        return c.HTML(404, "404.html", nil)
    })),
    
    // Override the global error catcher for 500s and other HTTP errors
    mux.WithErrorHandler(func(c core.Context, err error) {
        // Extract the code (defaults to 500 if it's a standard error)
        code := http.StatusInternalServerError
        if he, ok := err.(*httperrors.HTTPError); ok {
            code = he.Code
        }

        // Render your custom error template
        _ = c.HTML(code, "error.html", map[string]any{
            "Code":  code,
            "Error": err.Error(),
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

---

## 5. HTML Templating

By default, `httpex` uses the `JSONRenderer`. If you are building a server-side rendered application, inject the `HTMLRenderer` from the `renderer` package:

```go
import (
    "html/template"
    "github.com/isaacjuwon/httpex/pkg/mux"
    "github.com/isaacjuwon/httpex/pkg/renderer"
)

tmpl := template.Must(template.ParseGlob("views/*.html"))

m := mux.New(
    mux.WithRenderer(&renderer.HTMLRenderer{
        Templates: tmpl,
    }),
)

m.Get("/", func(c core.Context) error {
    // Automatically executes "index.html" with the map data
    return c.HTML(200, "index.html", map[string]string{
        "Title": "Welcome!",
    })
})
```

---

## 6. Middlewares

The `pkg/middleware` package provides essential protections. They are highly optimized and secure by default. 

```go
m.Use(
    middleware.RequestID(),            // Sets X-Request-Id header
    middleware.Logging(),              // Structured logs
    middleware.Recovery(),             // Prevents full server crashes on panic
    middleware.SecureHeaders(),        // HSTS, XSS protections, framing limits
    middleware.BodyLimit(2 * 1024 * 1024), // Max 2MB incoming JSON
    middleware.Timeout(5 * time.Second),   // Context cancellation
    middleware.CORS(),                 // Flexible Cross-Origin rules
)
```

---

## 7. Configuration & Options

`httpex` is configured explicitly via functional options passed to `mux.New()`. 

- `WithRenderer(core.Renderer)`
- `WithErrorHandler(errors.ErrorHandler)`
- `WithRouter(core.Router)`
- `WithNotFound(core.Handler)`
- `WithMethodNotAllowed(core.Handler)`

```go
// Custom 404 handler
m := mux.New(
    mux.WithNotFound(core.HandlerFunc(func(c core.Context) error {
        return c.JSON(404, map[string]string{"error": "where are you going?"})
    })),
)
```
