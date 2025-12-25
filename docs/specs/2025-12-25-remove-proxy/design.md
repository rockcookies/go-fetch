# Remove Proxy and HTTPTransport

## Overview

Remove internal Proxy management logic and `HTTPTransport()` type assertion method. Proxy configuration should be handled by users directly via `http.Transport`.

## Goals

- Remove `Client.proxyURL` field
- Remove `ProxyURL()`, `SetProxy()`, `RemoveProxy()`, `IsProxySet()` methods
- Remove `HTTPTransport()` method
- Change `SetTransport()` to strict mode (panic on nil)
- Remove `ErrNotHttpTransportType` error variable

## Changes

### client.go

**Remove field (line 110):**
```go
proxyURL *url.URL
```

**Remove methods:**
- `ProxyURL() *url.URL` (lines 650-655)
- `SetProxy(proxyURL string) *Client` (lines 657-676)
- `RemoveProxy() *Client` (lines 678-691)
- `HTTPTransport() (*http.Transport, error)` (lines 708-716)
- `IsProxySet() bool` (lines 878-881)

**Modify SetTransport:**
```go
// SetTransport sets a custom http.RoundTripper.
// NOTE: It overwrites the existing transport.
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if transport == nil {
		panic("SetTransport: transport cannot be nil")
	}
	c.httpClient.Transport = transport
	return c
}
```

**Modify Clone:**
Remove proxyURL cloning logic (lines 908-910).

**Remove variable:**
```go
ErrNotHttpTransportType = errors.New("resty: not a http.Transport type")
```

## Tests

### client_test.go

**Remove tests:**
- `TestClient_ProxyURL`
- `TestClient_SetProxy`
- `TestClient_RemoveProxy`
- `TestClient_IsProxySet`
- `TestClient_HTTPTransport`

**Add:**
```go
func TestClient_SetTransport_NilPanic(t *testing.T) {
	c := New()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("SetTransport(nil) should panic")
		}
	}()
	c.SetTransport(nil)
}
```

**Modify:**
- `TestClient_Clone` - remove proxyURL assertions
- `TestClient_Transport` - only test getter

## Migration Guide

**Before:**
```go
client.SetProxy("http://proxy.example.com:8080")
client.RemoveProxy()

transport, _ := client.HTTPTransport()
transport.TLSClientConfig = &tls.Config{...}
```

**After:**
```go
proxyURL, _ := url.Parse("http://proxy.example.com:8080")
transport := &http.Transport{
    Proxy: http.ProxyURL(proxyURL),
}

// Use constructor
client := NewWithTransport(transport)

// Or replace
client.SetTransport(transport)
```

## Impact

- ~60-80 lines removed
- ~5 lines modified
- ~10 lines added (tests)
- Net reduction: ~50-70 lines

**Breaking Change:** Users must create `http.Transport` directly instead of using proxy methods.

## Constitution Compliance

| Principle | Status |
|-----------|--------|
| 1.1 YAGNI | Remove unnecessary abstraction |
| 1.2 Standard Library First | Use `http.Transport` directly |
| 1.3 Anti-Over-Engineering | 5 methods → 0 |
| 2.1 TDD | Tests first |
| 3.1 Error Handling | Panic for invalid input |
| 3.2 No Global Variables | No change |
| 4.1 Package Cohesion | Clearer responsibilities |
