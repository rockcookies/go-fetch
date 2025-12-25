# Simplify Transport Initialization

## Overview

Remove internal Transport creation logic, let users provide `http.RoundTripper` directly.

## Background

Currently has 4 Transport-related constructors that internally create `*http.Transport` via `TransportSettings` struct. This adds unnecessary complexity.

## Goals

- Remove `NewWithTransportSettings`, `NewWithDialer`, `NewWithLocalAddr`, `NewWithDialerAndTransportSettings`
- Remove `TransportSettings` struct and `createTransport` function
- Remove `transport_dial.go` and `transport_dial_wasm.go` files
- Add single `NewWithTransport(http.RoundTripper)` function
- If transport is nil, use `http.DefaultTransport`

## Changes

### resty.go

**Remove:**
- `NewWithTransportSettings(*TransportSettings)`
- `NewWithDialer(*net.Dialer)`
- `NewWithLocalAddr(net.Addr)`
- `NewWithDialerAndTransportSettings(*net.Dialer, *TransportSettings)`
- `createTransport(*net.Dialer, *TransportSettings)`

**Add:**
```go
// NewWithTransport creates a new Resty client with the given [http.RoundTripper].
// If transport is nil, [http.DefaultTransport] is used.
func NewWithTransport(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return NewWithClient(&http.Client{
		Jar:       createCookieJar(),
		Transport: transport,
	})
}
```

**Modify:**
```go
func New() *Client {
	return NewWithTransport(nil)
}
```

### client.go

**Remove:** `TransportSettings` struct (lines 99-143)

### Delete Files

- `transport_dial.go`
- `transport_dial_wasm.go`

### Tests

| Test | Action |
|------|--------|
| `TestClientTransportSettings` (custom) | Remove or use `NewWithTransport` + manual `*http.Transport` |
| `TestClientTransportSettings` (default subtests) | Use `New()` or `NewWithTransport(nil)` |
| `TestNewWithDialer` | **Delete entire test** |
| `TestNewWithLocalAddr` | **Delete entire test** |
| `TestTraceInfoOnTimeout` | Use `NewWithTransport` + manual `*http.Transport` with timeout |

**Add:**
- `TestNewWithTransport` - verify nil uses `http.DefaultTransport`
- `TestNewWithTransport_Custom` - verify custom transport is used

## Impact

- ~150-200 lines removed
- ~10 lines added
- Net reduction: ~140-190 lines

## Constitution Compliance

| Principle | Status |
|-----------|--------|
| 1.1 YAGNI | ✅ Remove unnecessary complexity |
| 1.2 Standard Library First | ✅ Use `http.Transport` directly |
| 1.3 Anti-Over-Engineering | ✅ 4 constructors → 1 |
| 2.1 TDD | ✅ Tests first |
| 3.1 Error Handling | ✅ No error return scenarios |
| 3.2 No Global Variables | ✅ Explicit dependency injection |
| 4.1 Package Cohesion | ✅ Clearer responsibilities |
