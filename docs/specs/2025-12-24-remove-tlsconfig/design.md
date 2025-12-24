# Remove TLSConfig Feature

## Summary

Remove the TLSClientConfig-related functionality from the go-fetch library. This includes the `TLSClientConfiger` interface, `TLSClientConfig()`, `SetTLSClientConfig()`, and `tlsConfig()` methods, along with their tests.

## Motivation

Following the project's simplification initiative (see `docs/specs/2025-12-24-further-simplify`), we are removing non-core features to maintain a lean, focused HTTP client. TLS configuration is a transport-level concern that users can handle by:

1. Creating a custom `http.Transport` with desired TLS settings
2. Using `client.SetTransport()` to set the custom transport

This aligns with Principle 1.1 (YAGNI) - the library should not provide abstraction for functionality that the standard library already handles well.

## Design

### Components to Remove

**From `client.go`:**

1. `TLSClientConfiger` interface (lines 94-99):
   ```go
   TLSClientConfiger interface {
       TLSClientConfig() *tls.Config
       SetTLSClientConfig(*tls.Config) error
   }
   ```

2. `TLSClientConfig()` method (lines 1032-1040):
   ```go
   func (c *Client) TLSClientConfig() *tls.Config
   ```

3. `SetTLSClientConfig()` method (lines 1042-1073):
   ```go
   func (c *Client) SetTLSClientConfig(tlsConfig *tls.Config) *Client
   ```

4. `tlsConfig()` private method (lines 1627-1645):
   ```go
   func (c *Client) tlsConfig() (*tls.Config, error)
   ```

5. Update `SetTransport()` documentation comment to remove reference to `TLSClientConfiger` (line 1180)

**From `client_test.go`:**

1. `CustomRoundTripper2` test struct and its methods (lines 142-167)
2. `TestClientTLSConfigerInterface` test function (lines 169-211)
3. Other usages of `SetTLSClientConfig()` in tests (lines 321, 836)

**From `request_test.go`:**

1. Usages of `SetTLSClientConfig()` (lines 1340, 1701)

### User Migration Path

Users who need custom TLS configuration should do:

```go
// Old way (to be removed):
client.SetTLSClientConfig(&tls.Config{
    InsecureSkipVerify: true,
})

// New way:
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true,
    },
}
client.SetTransport(transport)
```

## Affected Code

- `client.go`: ~80 lines removed
- `client_test.go`: ~70 lines removed
- `request_test.go`: 2 lines removed

## Testing Strategy

1. Remove all tests for removed functionality
2. Ensure existing tests still pass after removal
3. Verify that `SetTransport()` still works for users who want custom TLS

## Risks

- Breaking change for existing users of `SetTLSClientConfig()` / `TLSClientConfig()`
- Mitigation: Clear migration path documentation

## Checklist

- [ ] Remove `TLSClientConfiger` interface
- [ ] Remove `TLSClientConfig()` method
- [ ] Remove `SetTLSClientConfig()` method
- [ ] Remove `tlsConfig()` private method
- [ ] Update `SetTransport()` documentation
- [ ] Remove `CustomRoundTripper2` test struct
- [ ] Remove `TestClientTLSConfigerInterface` test
- [ ] Remove other test usages
- [ ] Run `go test ./...`
