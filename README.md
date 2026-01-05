# go-fetch

A composable HTTP client for Go built on `net/http` with middleware support.

## Why go-fetch?

**Simplicity is the ultimate sophistication.** No frameworks, no magic, no unnecessary abstractions.

- Pure `net/http` - zero non-standard dependencies
- Middleware composition for clean request/response handling
- Explicit error handling - never ignore errors
- Table-driven tests with high coverage

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
    dispatcher := fetch.NewDispatcher(nil)

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

Wraps `http.Client` and manages middleware chains. Thread-safe.

```go
dispatcher := fetch.NewDispatcher(nil) // 30s default timeout

// Custom client
client := &http.Client{Timeout: 10 * time.Second}
dispatcher := fetch.NewDispatcher(client)

// Global middleware
dispatcher.Use(authMiddleware)
```

### Request

Accumulates middleware before execution:

```go
req := dispatcher.NewRequest()
req.Use(customMiddleware)
resp := req.JSON(map[string]string{"name": "John"}).Send("POST", url)
defer resp.Close()
```

### Middleware

Wraps handlers for cross-cutting concerns:

```go
authMiddleware := func(next fetch.Handler) fetch.Handler {
    return fetch.HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
        req.Header.Set("Authorization", "Bearer token")
        return next.Handle(client, req)
    })
}
```

## Features

### Request Body

**JSON:**
```go
resp := req.JSON(map[string]string{"name": "John"}).Send("POST", url)
defer resp.Close()
```

**XML:**
```go
resp := req.XML(User{Name: "John"}).Send("POST", url)
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
resp := req.Body(strings.NewReader("data")).Send("POST", url)
defer resp.Close()
```

**Lazy Body:**
```go
resp := req.BodyGet(func() (io.Reader, error) {
    return loadDataFromFile() // Computed only when needed
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

### Response Handling

```go
resp := req.Send("GET", url)
defer resp.Close()  // Always defer - safe with errors

if resp.Error != nil {
    return fmt.Errorf("request failed: %w", resp.Error) // Explicit wrapping
}

// Access response
fmt.Println(resp.RawResponse.StatusCode)
body := resp.String()

// Decode JSON
var result MyStruct
if err := resp.JSON(&result); err != nil {
    return fmt.Errorf("decode failed: %w", err)
}
```

### Headers & Cookies

```go
// Headers
dispatcher.Use(fetch.HeaderFuncs(func(h http.Header) {
    h.Set("User-Agent", "MyApp/1.0")
}))

// Cookies
dispatcher.Use(fetch.CookiesAdd(&http.Cookie{
    Name:  "session",
    Value: "token123",
}))
```

### Client Configuration

```go
dispatcher.Use(fetch.ClientFuncs(func(c *http.Client) {
    c.Timeout = 10 * time.Second
    c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse // Disable redirects
    }
}))
```

## Advanced Usage

### Request Cloning

```go
baseReq := dispatcher.NewRequest()
baseReq.UseFuncs(func(r *http.Request) {
    r.Header.Set("Authorization", "Bearer token")
})

req1 := baseReq.Clone().Send("GET", "/users")
req2 := baseReq.Clone().Send("GET", "/posts")
```

### Request Dumping

```go
import "github.com/rockcookies/go-fetch/dump"

dispatcher.Use(dump.Middleware()) // Debug all requests/responses

// With filters
dispatcher.Use(dump.Middleware(dump.WithFilter(dump.SkipResponseBody())))
```

## Development Philosophy

Follows strict principles:

1. **Simplicity First (YAGNI)** - Only essential features, no over-engineering
2. **Standard Library First** - Built on `net/http`, avoid external dependencies
3. **Explicit Over Implicit** - Never ignore errors, always wrap with context
4. **Single Responsibility** - Each component does one thing well
5. **Test-First Imperative** - TDD with table-driven tests (Red-Green-Refactor)

Read [docs/constitution.md](docs/constitution.md) for complete development constitution.

## Testing

```bash
go test ./...           # Run all tests
go test -v -race ./...  # With race detector
go test -cover ./...    # Coverage report
```

All code follows TDD: write failing test → make it pass → refactor.

## License

See [LICENSE](LICENSE).

## Contributing

Pull requests must:

- Start with a failing test (TDD)
- Keep it simple (no abstractions without clear need)
- Use standard library when possible
- Handle all errors explicitly (never `_` errors)
- Add table-driven tests for edge cases

Constitutional review: all changes must comply with [docs/constitution.md](docs/constitution.md).
