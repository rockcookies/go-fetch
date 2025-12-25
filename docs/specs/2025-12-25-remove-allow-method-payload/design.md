# Remove AllowMethodGetPayload/AllowMethodDeletePayload

## Overview

Remove the `AllowMethodGetPayload` and `AllowMethodDeletePayload` functionality from the codebase. As a simple HTTP client library, these restrictions should be controlled by users, not enforced by the library.

## Rationale

HTTP specifications do not prohibit GET or DELETE methods from having a request body. These restrictions are an artificial limitation that adds unnecessary complexity. Users who need GET/DELETE with body should be able to do so without library-imposed warnings or restrictions.

## Changes

### client.go

Remove fields:
- `allowMethodGetPayload bool`
- `allowMethodDeletePayload bool`

Remove methods:
- `AllowMethodGetPayload() bool`
- `SetAllowMethodGetPayload(allow bool) *Client`
- `AllowMethodDeletePayload() bool`
- `SetAllowMethodDeletePayload(allow bool) *Client`

Remove from `Clone()`:
- Copy logic for `allowMethodGetPayload` and `allowMethodDeletePayload`

Remove from `Client.values` struct (if present):
- `AllowMethodGetPayload` field
- `AllowMethodDeletePayload` field

### request.go

Remove fields:
- `AllowMethodGetPayload bool`
- `AllowMethodDeletePayload bool`

Remove methods:
- `SetAllowMethodGetPayload(allow bool) *Request`
- `SetAllowMethodDeletePayload(allow bool) *Request`

Remove from `buildHTTPRequest()`:
- Conditional checks for `AllowMethodGetPayload` and `AllowMethodDeletePayload` (lines ~1335-1340)

### Tests

**client_test.go:**
- Remove `TestClientAllowMethodGetPayload` (lines ~337-380)
- Remove `TestClientAllowMethodDeletePayload` (lines ~382-421)

**middleware_test.go:**
- Remove test case: "string body with GET method and AllowMethodGetPayload by client"
- Remove test case: "string body with GET method and AllowMethodGetPayload by request"
- Remove test case: "string body with DELETE method with AllowMethodDeletePayload by request"

**request_test.go:**
- Remove test cases using `SetAllowMethodGetPayload(true)`
- Remove test cases using `SetAllowMethodDeletePayload(true)`

**stream_test.go:**
- Clean up reference to `SetAllowMethodGetPayload(true)` in comments

## Implementation Strategy

1. Remove logic from `request.go` (core change)
2. Remove fields and methods from `request.go` and `client.go`
3. Clean up all tests
4. Run `go test ./...` to verify no compilation errors
5. Run `go build` to verify build succeeds

## Verification

- Code compiles without errors
- All tests pass
- GET/DELETE requests can have body without warnings

## Impact

- Approximately 150-200 lines of code removed
- Simpler API with fewer concepts
- Breaking change for users calling these methods, but the functionality still works (just without the restrictions)
