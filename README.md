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
    "context"
    "fmt"
    "github.com/rockcookies/go-fetch"
)

func main() {
    // Create a dispatcher (HTTP client wrapper)
    dispatcher := fetch.NewDispatcher(nil)

    // Make a simple GET request
    resp := dispatcher.NewRequest().Get(context.Background(), "https://api.example.com/users")
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

The `Request` type accumulates request formatters and middleware before execution:

```go
ctx := context.Background()
req := dispatcher.NewRequest()
req.Use(customMiddleware)
req.JSON(map[string]string{"name": "John"})
resp := req.Post(ctx, "https://api.example.com/users")
```

Request formatters and middleware follow symmetric prepend/append APIs:

| Middleware | Funcs |
|------------|-------|
| `Pre(...)` — prepend | `PreFuncs(...)` — prepend |
| `Use(...)` — append | `UseFuncs(...)` — append |

Formatters run in `Do` before middleware. All HTTP methods require `context.Context` as the first argument.

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
resp := req.JSON(data).Post(context.Background(), url)
defer resp.Close()
```

**XML:**
```go
data := User{Name: "John"}
resp := req.XML(data).Post(context.Background(), url)
defer resp.Close()
```

**Form:**
```go
form := url.Values{}
form.Set("username", "john")
resp := req.Form(form).Post(context.Background(), url)
defer resp.Close()
```

**Raw Body:**
```go
reader := strings.NewReader("raw data")
resp := req.Body(reader).Post(context.Background(), url)
defer resp.Close()
```

**Lazy Body:**
```go
resp := req.BodyGet(func() (io.Reader, error) {
    // Body is only computed when needed
    return loadDataFromFile()
}).Post(context.Background(), url)
defer resp.Close()
```

### Multipart Forms

```go
fields := []*fetch.MultipartField{
    {Name: "file", FileName: "doc.txt", Content: fileReader},
    {Name: "description", Value: "My file"},
}
resp := req.Multipart(fields).Post(context.Background(), url)
defer resp.Close()
```

### URL Building

```go
resp := req.Get(context.Background(), "https://api.example.com/search?q=go")
defer resp.Close()

// Or use helper
resp := req.Get(context.Background(), fetch.BuildURL("https://api.example.com/search",
    fetch.WithQuery("q", "go"),
    fetch.WithQuery("limit", "10"),
))
defer resp.Close()
```

### Response Handling

```go
resp := req.Get(context.Background(), url)
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
req.UseFuncs(func(r *http.Request) *http.Request {
    r.Header.Set("User-Agent", "MyApp/1.0")
    r.Header.Set("Accept", "application/json")
    return nil
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
req.PreFuncs(func(r *http.Request) *http.Request {
    return r.WithContext(ctx)
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
req.PreFuncs(func(r *http.Request) *http.Request {
    return r.WithContext(ctx)
})
```

## Advanced Usage

### Cloning Requests

```go
ctx := context.Background()
baseReq := dispatcher.NewRequest()
baseReq.PreFuncs(func(r *http.Request) *http.Request {
    r.Header.Set("Authorization", "Bearer token")
    return nil
})

// Clone for different endpoints
req1 := baseReq.Clone().Get(ctx, "/users")
req2 := baseReq.Clone().Get(ctx, "/posts")
```

### Request Dumping

The `dump` package provides `dump.Transport`, an `http.RoundTripper` wrapper built with `dump.New` that captures HTTP exchanges and forwards them to a `DumpWriter`.

```go
import (
    "log/slog"
    "net/http"

    fetch "github.com/rockcookies/go-fetch"
    "github.com/rockcookies/go-fetch/dump"
)

client := &http.Client{
    Transport: dump.New(
        http.DefaultTransport,
        dump.WithWriter(&dump.SlogWriter{
            Logger: slog.Default(),
            Level:  slog.LevelInfo,
        }),
        dump.WithOptions(dump.DumpOptions{
            RequestHeaders:  true,
            RequestBody:     true,
            ResponseHeaders: true,
            ResponseBody:    true,
        }),
    ),
}
dispatcher := fetch.NewDispatcher(client)
```

**DumpOptions** — boolean flags select what is captured (metadata is always included):

| Field | Description |
|-------|-------------|
| `RequestHeaders` / `RequestBody` | Request headers and body |
| `ResponseHeaders` / `ResponseBody` | Response headers and body (tee capture; **must** read and `Close` `resp.Body` to flush the dump — omitting Close drops the entry) |
| `BodyMaxBytes` | Cap per body; `0` → 64 KiB; `-1` → unlimited |
| `SkipBinaryBody` | Skip body capture when Content-Type is not text-like |
| `DumpOnError` | Emit a dump entry when RoundTrip returns an error (`Meta.Err` is set; `resp` is nil) |

**Latency** — `Meta.Latency` is measured until the dump is written. With `ResponseBody` enabled, that includes reading through `Close` (not TTFB alone).

**Streaming request bodies** — capture uses `req.GetBody` when present; if `GetBody` is nil (e.g. `io.Pipe` multipart upload), the body is not read and `Meta.ReqBodySkipped` is set.

**Filtering** — `WithFilter` runs after a successful RoundTrip (status, URL, headers). Returning `false` skips the dump. Use `WithEntryFilter` for rules that need captured body bytes (JSON fields, sampling).

```go
dump.New(http.DefaultTransport,
    dump.WithWriter(&dump.SlogWriter{Logger: slog.Default()}),
    dump.WithOptions(dump.DumpOptions{
        RequestHeaders: true,
        ResponseHeaders: true,
    }),
    // Dump only 4xx/5xx responses
    dump.WithFilter(dump.StatusFilter([2]int{400, 599})),
)
```

Available filters: `URLFilter`, `MethodFilter`, `StatusFilter`, `HeaderFilter`, `AllFilters`, `AnyFilter`, `NotFilter`.

**Writers** — `SlogWriter` (structured slog; body attributes capped at 2 KiB for log volume), `IOWriter` (human-readable text), `MultiWriter` (fan-out), `NoopWriter` (discard). `DumpWriter.Write` runs on the RoundTrip goroutine — wrap with a buffered channel + background worker if the sink is slow (Elasticsearch, Kafka, etc.).

**Entry filter** — `WithEntryFilter(func(DumpEntry) bool)` runs after capture, before write. Return `false` to drop the entry (e.g. filter by `RespBody`, or sample with `rand`). Implement `DumpWriter` for other custom sinks.

**Redaction** — request and response use separate `Redactor` instances (`WithRequestRedactor`, `WithResponseRedactor`). Sensitive values in the dump entry are replaced with `[REDACTED]`; live `req`/`resp` headers are not modified.

```go
reqRedact := dump.DefaultRedactor{
    Headers: map[string]struct{}{
        "authorization": {},
        "x-api-key":     {},
        "cookie":        {},
    },
}
respRedact := dump.DefaultRedactor{
    Headers: map[string]struct{}{
        "set-cookie": {},
    },
}

dump.New(http.DefaultTransport,
    dump.WithWriter(&dump.SlogWriter{Logger: slog.Default(), Level: slog.LevelInfo}),
    dump.WithOptions(dump.DumpOptions{RequestHeaders: true, ResponseHeaders: true}),
    dump.WithRequestRedactor(reqRedact),
    dump.WithResponseRedactor(respRedact),
)
```

Header blocklist keys are lowercase. The same `DefaultRedactor` value may be passed to both options when the blocklist is identical.

**Tracing IDs** — `WithMetaExtractor` pulls `TraceID` and `ReqID` from `req.Context()` into `DumpMeta`.

### Error Handling

All errors follow explicit handling patterns:

```go
resp := req.Get(context.Background(), url)
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
