# Simplify URL Parameters Handling

## Overview

Simplify URL parameter handling by removing complex PathParams replacement logic and the non-standard `unescapeQueryParams` feature.

## Goals

1. Replace complex PathParams placeholder replacement with simple `strings.ReplaceAll`
2. Remove escape handling distinction (SetPathParam vs SetRawPathParam)
3. Remove `unescapeQueryParams` functionality
4. Reduce code complexity by ~50-60 lines

## Changes

### 1. PathParams Simplification (middleware.go)

**Current**: Lines 40-94 use buffer pool, manual string traversal, edge case handling for `{}`, unclosed braces, missing params.

**New**: Simple `replacePlaceholders` function:

```go
// replacePlaceholders replaces {key} with values from params
func replacePlaceholders(url string, params map[string]string) string {
    for key, value := range params {
        placeholder := "{" + key + "}"
        url = strings.ReplaceAll(url, placeholder, value)
    }
    return url
}
```

**Updated parseRequestURL**:

```go
// Merge client and request path params
if len(c.PathParams())+len(r.PathParams) > 0 {
    for p, v := range c.PathParams() {
        if _, ok := r.PathParams[p]; !ok {
            r.PathParams[p] = v
        }
    }
    r.URL = replacePlaceholders(r.URL, r.PathParams)
}
```

### 2. Remove unescapeQueryParams

**Delete**:
- `middleware.go:143-150` - unescape logic
- `request.go:64` - `unescapeQueryParams bool` field
- `request.go:415-420` - `SetUnescapeQueryParams` method
- `client.go:111` - `unescapeQueryParams bool` field
- `client.go:799-806` - `SetUnescapeQueryParams` method
- `client.go:294` - field initialization

**Keep**: QueryParams using `url.Values.Encode()` unchanged.

### 3. Simplify PathParams API

**Delete methods**:
- `SetPathParam` (with PathEscape)
- `SetPathParams` (with PathEscape)
- `SetRawPathParam` (no escape)
- `SetRawPathParams` (no escape)

**Replace with unified API**:

```go
// SetPathParam sets a single URL path parameter (replaces {key} in URL).
func (r *Request) SetPathParam(param, value string) *Request {
    r.PathParams[param] = value
    return r
}

// SetPathParams sets multiple URL path parameters.
func (r *Request) SetPathParams(params map[string]string) *Request {
    for p, v := range params {
        r.PathParams[p] = v
    }
    return r
}
```

Same for `Client`.

## Testing

### Update Tests

- `middleware_test.go` - Remove PathParams/unescape edge case tests
- `request_test.go` - Update/remove `TestGetPathParamAndPathParams`, `TestPathParamURLInput`, `TestRawPathParamURLInput`
- `benchmark_test.go` - Update `Benchmark_parseRequestURL_PathParams`

### New Test Cases

```go
func TestReplacePlaceholders(t *testing.T) {
    tests := []struct {
        url    string
        params map[string]string
        want   string
    }{
        {"/users/{id}", map[string]string{"id": "123"}, "/users/123"},
        {"/users/{id}/posts/{postId}", map[string]string{"id": "123", "postId": "456"}, "/users/123/posts/456"},
        {"/users/{id}", map[string]string{}, "/users/{id}"},
        {"/users/{id}/profile", map[string]string{"id": "user/name"}, "/users/user/name/profile"},
    }
    // ...
}
```

## Trade-offs

### Benefits
- Simpler code, easier to understand
- ~50-60 lines of complex logic removed
- Single unified API for PathParams

### Costs
- Users must ensure parameter values are correctly formatted (no auto-escape)
- Slight performance decrease (multiple ReplaceAll vs single traversal) - negligible for most use cases

## Migration Guide

| Old API | New API | Notes |
|---------|---------|-------|
| `SetRawPathParam(k, v)` | `SetPathParam(k, v)` | Direct replacement |
| `SetPathParam(k, v)` | `SetPathParam(k, url.PathEscape(v))` | User must escape if needed |
| `SetUnescapeQueryParams(true)` | N/A | Feature removed |

## Files Modified

- `middleware.go` - Simplify parseRequestURL, remove unescape
- `request.go` - Remove SetRawPath* methods, remove unescape field/method
- `client.go` - Remove SetRawPath* methods, remove unescape field/method
- `middleware_test.go` - Update tests
- `request_test.go` - Update tests
- `benchmark_test.go` - Update benchmarks
