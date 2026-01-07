# go-fetch

Composable HTTP client built on pure `net/http` with middleware support.

## Philosophy

**Less is more.** Zero non-standard dependencies, no magic, explicit errors.

- Pure `net/http` - standard library only
- Middleware composition - clean separation of concerns
- Explicit error handling - never ignore, always wrap
- TDD with table-driven tests

## Installation

```bash
go get github.com/rockcookies/go-fetch
```

## Quick Start

```go
dispatcher := fetch.NewDispatcher(nil)

resp := dispatcher.R().
    JSON(map[string]string{"name": "John"}).
    Send("POST", "https://api.example.com/users")
defer resp.Close()

if resp.Error != nil {
    return fmt.Errorf("request failed: %w", resp.Error)
}

var user User
if err := resp.JSON(&user); err != nil {
    return fmt.Errorf("decode failed: %w", err)
}
```

## Core Concepts

### Dispatcher

Wraps `http.Client` with middleware chains:

```go
// Default client
dispatcher := fetch.NewDispatcher(nil)

// Custom client
client := &http.Client{Timeout: 10 * time.Second}
dispatcher := fetch.NewDispatcher(client)

// Global middleware
dispatcher.Use(authMiddleware)
```

### Request

Chain-able request builder:

```go
resp := dispatcher.R().
    HeaderKV("Authorization", "Bearer token").
    AddCookie(&http.Cookie{Name: "session", Value: "abc"}).
    JSON(payload).
    Send("POST", url)
defer resp.Close()
```

### Middleware

Standard `net/http` handler pattern:

```go
func authMiddleware(next fetch.Handler) fetch.Handler {
    return fetch.HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
        req.Header.Set("Authorization", "Bearer "+getToken())
        return next.Handle(client, req)
    })
}
```

## API Reference

### Request Body

```go
// JSON/XML
req.JSON(data)              // encoding/json
req.XML(data)               // encoding/xml

// Form
req.Form(url.Values{})      // application/x-www-form-urlencoded

// Raw
req.Body(reader)            // io.Reader
req.BodyGet(func() (io.Reader, error) {...})     // lazy evaluation
req.BodyGetBytes(func() ([]byte, error) {...})   // lazy bytes

// Multipart
req.Multipart([]*fetch.MultipartField{...})
```

### Headers & Cookies

```go
// Headers
req.HeaderKV("Content-Type", "application/json")   // set single
req.AddHeaderKV("Accept", "text/html")             // append single
req.HeaderFromMap(map[string]string{...})          // set batch
req.Header(func(h http.Header) {...})              // custom logic

// Cookies
req.AddCookie(&http.Cookie{...})    // append
req.DelAllCookies()                 // clear
```

### Response

```go
resp := req.Send("GET", url)
defer resp.Close()  // ALWAYS defer

if resp.Error != nil {
    return fmt.Errorf("failed: %w", resp.Error)  // NEVER ignore
}

// Access
statusCode := resp.RawResponse.StatusCode
body := resp.String()
bytes := resp.Bytes()

// Decode
var result T
if err := resp.JSON(&result); err != nil {
    return fmt.Errorf("decode: %w", err)
}
```

### Client Tuning

```go
dispatcher.Use(fetch.ClientFuncs(func(c *http.Client) {
    c.Timeout = 5 * time.Second
    c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse  // disable redirects
    }
}))
```

## Testing

```bash
go test ./...           # all tests
go test -v -race ./...  # with race detector
go test -cover ./...    # coverage
```

TDD cycle: Red (write failing test) → Green (make it pass) → Refactor.

## Contributing

All contributions must follow [docs/constitution.md](docs/constitution.md):

1. **YAGNI** - implement only what's needed
2. **Standard Library First** - `net/http` over frameworks
3. **Explicit Errors** - never `_`, always wrap with `%w`
4. **TDD** - failing test first, table-driven style
5. **Single Responsibility** - one thing, done well

## License

MIT - see [LICENSE](LICENSE).
