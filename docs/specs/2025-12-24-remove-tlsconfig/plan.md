# Implementation Plan: Remove TLSConfig

## Overview
Remove TLSClientConfig-related functionality from go-fetch, approximately 150 lines of code.

## Tasks

### Phase 1: Remove from client.go (~80 lines)

- [ ] 1.1 Remove `TLSClientConfiger` interface (lines 94-99)
- [ ] 1.2 Remove `TLSClientConfig()` method (lines 1032-1040)
- [ ] 1.3 Remove `SetTLSClientConfig()` method (lines 1042-1073)
- [ ] 1.4 Remove `tlsConfig()` private method (lines 1627-1645)
- [ ] 1.5 Update `SetTransport()` documentation comment to remove `TLSClientConfiger` reference (line 1180)

### Phase 2: Remove from client_test.go (~70 lines)

- [ ] 2.1 Remove `CustomRoundTripper2` struct definition and methods (lines 142-167)
- [ ] 2.2 Remove `TestClientTLSConfigerInterface` test function (lines 169-211)
- [ ] 2.3 Remove `SetTLSClientConfig` call at line 321
- [ ] 2.4 Remove `SetTLSClientConfig` call at line 836

### Phase 3: Remove from request_test.go (2 lines)

- [ ] 3.1 Remove `SetTLSClientConfig` call at line 1340
- [ ] 3.2 Remove `SetTLSClientConfig` call at line 1701

### Phase 4: Verify

- [ ] 4.1 Run `go test ./...` to ensure all tests pass
- [ ] 4.2 Run `go build ./...` to ensure no compilation errors

## Migration Guide for Users

```go
// Before (removed):
client.SetTLSClientConfig(&tls.Config{
    InsecureSkipVerify: true,
})

// After:
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true,
    },
}
client.SetTransport(transport)
```

## Summary

- **Files**: 3 (client.go, client_test.go, request_test.go)
- **Lines removed**: ~150
- **Breaking change**: Yes - `TLSClientConfig()` and `SetTLSClientConfig()` methods removed
