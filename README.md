# go-fetch

A simple, composable HTTP client for Go with middleware support.

## Philosophy

**Simple is better than complex.** This library follows Go's "less is more" philosophy:

- Built on `net/http` standard library
- No unnecessary abstractions or dependencies
- Middleware-based composition for flexibility
- Explicit error handling throughout

## Installation

```bash
go get github.com/rockcookies/go-fetch
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/rockcookies/go-fetch"
)

func main() {
    // Create a dispatcher (HTTP client wrapper)
    dispatcher := fetch.NewDispatcher(nil)

    // Make a simple GET request
    resp := dispatcher.NewRequest().Send("GET", "https://api.example.com/users")
    defer resp.Close()

    if resp.Error != nil {
        fmt.Printf("Error: %v\n", resp.Error)
        return
    }

    fmt.Printf("Status: %d\n", resp.RawResponse.StatusCode)
    fmt.Printf("Body: %s\n", resp.String())
}
```

## Core Concepts

### Dispatcher

The `Dispatcher` wraps an `http.Client` and manages middleware chains. It's safe for concurrent use.

```go
// Create with default client (30s timeout)
dispatcher := fetch.NewDispatcher(nil)

// Or provide your own client
client := &http.Client{Timeout: 10 * time.Second}
dispatcher := fetch.NewDispatcher(client)

// Add global middleware
dispatcher.Use(middleware1, middleware2)
```

### Request

The `Request` type accumulates middleware before execution:

```go
req := dispatcher.NewRequest()
req.Use(customMiddleware)
req.JSON(map[string]string{"name": "John"})
resp := req.Send("POST", "https://api.example.com/users")
```

### Middleware

Middleware wraps handlers to add cross-cutting concerns:

```go
type Middleware func(Handler) Handler

// Example: Add authentication header
authMiddleware := func(next fetch.Handler) fetch.Handler {
    return fetch.HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
        req.Header.Set("Authorization", "Bearer token")
        return next.Handle(client, req)
    })
}

dispatcher.Use(authMiddleware)
```

## Features

### Body Encoding

**JSON:**
```go
data := map[string]string{"name": "John"}
resp := req.JSON(data).Send("POST", url)
defer resp.Close()
```

**XML:**
```go
data := User{Name: "John"}
resp := req.XML(data).Send("POST", url)
defer resp.Close()
```

**Form:**
```go
form := url.Values{}
form.Set("username", "john")
resp := req.Form(form).Send("POST", url)
defer resp.Close()
```

**Raw Body:**
```go
reader := strings.NewReader("raw data")
resp := req.Body(reader).Send("POST", url)
defer resp.Close()
```

**Lazy Body:**
```go
resp := req.BodyGet(func() (io.Reader, error) {
    // Body is only computed when needed
    return loadDataFromFile()
}).Send("POST", url)
defer resp.Close()
```

### Multipart Forms

```go
fields := []*fetch.MultipartField{
    {Name: "file", FileName: "doc.txt", Content: fileReader},
    {Name: "description", Value: "My file"},
}
resp := req.Multipart(fields).Send("POST", url)
defer resp.Close()
```

### URL Building

```go
resp := req.Send("GET", "https://api.example.com/search?q=go")
defer resp.Close()

// Or use helper
resp := req.Send("GET", fetch.BuildURL("https://api.example.com/search",
    fetch.WithQuery("q", "go"),
    fetch.WithQuery("limit", "10"),
))
defer resp.Close()
```

### Response Handling

```go
resp := req.Send("GET", url)
defer resp.Close()  // Always defer Close() - safe even when Error is present

// Check error first
if resp.Error != nil {
    return resp.Error
}

// Access response
fmt.Println(resp.RawResponse.StatusCode)
fmt.Println(resp.Header.Get("Content-Type"))

// Read body as string
body := resp.String()

// Or read as bytes
// bytes := resp.Bytes()

// Or decode JSON
// var result MyStruct
// err := resp.JSON(&result)
```

### Custom Headers and Options

```go
req.UseFuncs(func(r *http.Request) {
    r.Header.Set("User-Agent", "MyApp/1.0")
    r.Header.Set("Accept", "application/json")
})
```

### Headers Middleware

Configure headers at the dispatcher or request level using middleware:

```go
// Global header configuration
dispatcher.Use(fetch.PrepareHeaderMiddleware())
dispatcher.Use(fetch.SetHeaderOptions(func(opts *fetch.HeaderOptions) {
    opts.Header.Set("User-Agent", "MyApp/1.0")
    opts.Header.Set("Accept", "application/json")
}))

// Context-level headers
ctx := fetch.WithHeaderOptions(context.Background(), func(opts *fetch.HeaderOptions) {
    opts.Header.Set("Authorization", "Bearer token123")
})
req.UseFuncs(func(r *http.Request) {
    *r = *r.WithContext(ctx)
})
```

### Cookies Middleware

Manage cookies using middleware for consistent cookie handling:

```go
// Add cookies at the dispatcher level
dispatcher.Use(fetch.PrepareCookieMiddleware())
dispatcher.Use(fetch.SetCookieOptions(func(opts *fetch.CookieOptions) {
    opts.Cookies = append(opts.Cookies, &http.Cookie{
        Name:  "session",
        Value: "token123",
    })
}))

// Context-level cookies
ctx := fetch.WithCookieOptions(context.Background(), func(opts *fetch.CookieOptions) {
    opts.Cookies = append(opts.Cookies, &http.Cookie{
        Name:  "auth",
        Value: "secret",
    })
})
req.UseFuncs(func(r *http.Request) {
    *r = *r.WithContext(ctx)
})
```

## Advanced Usage

### Cloning Requests

```go
baseReq := dispatcher.NewRequest()
baseReq.UseFuncs(func(r *http.Request) {
    r.Header.Set("Authorization", "Bearer token")
})

// Clone for different endpoints
req1 := baseReq.Clone().Send("GET", "/users")
req2 := baseReq.Clone().Send("GET", "/posts")
```

### Request Dumping

The `dump` package provides middleware for debugging:

```go
import "github.com/rockcookies/go-fetch/dump"

// Dump all requests and responses
dispatcher.Use(dump.Middleware())

// Dump with filters
dispatcher.Use(dump.Middleware(
    dump.WithFilter(dump.SkipResponseBody()),
))
```

### Error Handling

All errors follow explicit handling patterns:

```go
resp := req.Send("GET", url)
defer resp.Close()  // Safe to defer immediately - handles all error cases

if resp.Error != nil {
    // Error is wrapped with context
    return fmt.Errorf("fetch failed: %w", resp.Error)
}

if resp.RawResponse.StatusCode >= 400 {
    // Handle HTTP errors
    return fmt.Errorf("HTTP error: %d", resp.RawResponse.StatusCode)
}
```

## Design Principles

This library strictly follows:

1. **Simplicity First** (YAGNI) - Only essential features
2. **Standard Library First** - Built on `net/http`
3. **Explicit Over Implicit** - No magic, clear error handling
4. **Single Responsibility** - Each component does one thing well

See [docs/constitution.md](docs/constitution.md) for the complete development philosophy.

## Testing

The library follows test-driven development with table-driven tests:

```bash
go test ./...
go test -v -race ./...
```

## License

See [LICENSE](LICENSE) file.

## Contributing

Contributions must follow the project constitution:

- Write tests first (TDD)
- Keep it simple (no over-engineering)
- Use standard library when possible
- Explicit error handling

See [docs/constitution.md](docs/constitution.md) for details.
