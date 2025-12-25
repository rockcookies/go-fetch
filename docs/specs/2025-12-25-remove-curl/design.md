# Design: Remove Curl Functionality

**Created:** 2025-12-25
**Status:** Proposed
**Rationale:** Debug curl functionality should be user-controlled via `DebugLogCallbackFunc`, not built into the library

## Overview

Remove all curl command generation and debug logging functionality from the codebase. Users who need curl commands can implement their own logic using `SetDebugLogCallback` or external tools.

## Affected Components

### Files to Delete
- `curl.go` (~100 lines) - curl command generation implementation
- `curl_test.go` (~270 lines) - all curl-related tests

### client.go Changes
**Fields to remove:**
- `generateCurlCmd bool`
- `debugLogCurlCmd bool`

**Methods to remove:**
- `EnableGenerateCurlCmd() *Client`
- `DisableGenerateCurlCmd() *Client`
- `SetGenerateCurlCmd(bool) *Client`
- `SetDebugLogCurlCmd(bool) *Client`

**Field propagation to remove:**
- `generateCurlCmd` from `requestClientValues` struct (line 412)
- `debugLogCurlCmd` from `requestClientValues` struct (line 413)

### request.go Changes
**Fields to remove:**
- `resultCurlCmd string`
- `generateCurlCmd bool`
- `debugLogCurlCmd bool`

**Methods to remove:**
- `EnableGenerateCurlCmd() *Request`
- `DisableGenerateCurlCmd() *Request`
- `SetGenerateCurlCmd(bool) *Request`
- `SetDebugLogCurlCmd(bool) *Request`
- `CurlCmd() string`
- `generateCurlCommand() string` (private)

### debug.go Changes
**RequestDebugLog struct - field to remove:**
```go
CurlCmd string `json:"curl_cmd"`
```

**Code blocks to remove:**
1. Lines 63-66: Conditional curl logging in debug output
   ```go
   if len(req.CurlCmd) > 0 {
       debugLog += "~~~ REQUEST(CURL) ~~~\n" +
           fmt.Sprintf("	%v\n", req.CurlCmd)
   }
   ```

2. Lines 164-166: Conditional assignment to debug log
   ```go
   if r.generateCurlCmd && r.debugLogCurlCmd {
       rdl.CurlCmd = r.resultCurlCmd
   }
   ```

### stream_test.go Changes
**Line 30:** Remove `.EnableGenerateCurlCmd()` call
```go
// Before:
client := New().EnableGenerateCurlCmd()

// After:
client := New()
```

## Breaking Changes

The following public APIs will be removed:

**Client-level methods:**
- `client.EnableGenerateCurlCmd()`
- `client.DisableGenerateCurlCmd()`
- `client.SetGenerateCurlCmd(bool)`
- `client.SetDebugLogCurlCmd(bool)`

**Request-level methods:**
- `request.EnableGenerateCurlCmd()`
- `request.DisableGenerateCurlCmd()`
- `request.SetGenerateCurlCmd(bool)`
- `request.SetDebugLogCurlCmd(bool)`
- `request.CurlCmd()`

## Migration Guide for Users

Users who need similar functionality can:

1. **Use `SetDebugLogCallback`:**
   ```go
   client.SetDebugLogCallback(func(r *Request, resp *Response) {
       // Access r.RawRequest and implement custom logging
       dump, _ := httputil.DumpRequestOut(r.RawRequest, true)
       fmt.Println(string(dump))
   })
   ```

2. **Use external tools:** mitmproxy, Charles, Fiddler, etc.

3. **Use `net/http/httputil`:**
   ```go
   dump, _ := httputil.DumpRequestOut(req.RawRequest, true)
   ```

## Implementation Steps

### Phase 1: Delete Core Files
1. Delete `curl.go`
2. Delete `curl_test.go`

### Phase 2: Clean Client
3. Remove fields from `Client` struct
4. Remove 4 methods from `client.go`
5. Remove field propagation in `requestClientValues`

### Phase 3: Clean Request
6. Remove fields from `Request` struct
7. Remove 6 methods from `request.go`

### Phase 4: Clean Debug Logic
8. Remove `CurlCmd` from `RequestDebugLog` struct
9. Remove curl logging blocks in `debug.go`

### Phase 5: Fix Tests
10. Remove `EnableGenerateCurlCmd()` call in `stream_test.go`

## Verification

**Build check:**
```bash
go build ./...
```

**Test check:**
```bash
go test ./... -v
```

**Static analysis:**
```bash
go vet ./...
```

**Dependency check:**
```bash
grep -r "EnableGenerateCurlCmd\|CurlCmd\|SetDebugLogCurlCmd" . --include="*.go"
```

## Impact

- **Lines removed:** ~150-200 (implementation + tests)
- **Files affected:** 6 files
- **API surface:** Reduced by 10 public methods
