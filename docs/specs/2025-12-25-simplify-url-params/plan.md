# Simplify URL Parameters Handling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** Simplify URL parameter handling by replacing complex PathParams replacement with simple string replacement and removing unescapeQueryParams functionality.

**Architecture:** Replace buffer-based manual string traversal with `strings.ReplaceAll` loop. Remove escape handling distinction. Remove non-standard unescape feature.

**Tech Stack:** Go standard library (`strings`, `net/url`)

---

## Constitutionality Review

| Principle | Status | Notes |
|-----------|--------|-------|
| Simplicity First | ✅ PASS | Removes ~50-60 lines of complex logic |
| Test-First | ✅ PASS | All changes start with failing tests |
| Clarity | ✅ PASS | Simpler code, easier to understand |
| Single Responsibility | ✅ PASS | Each function does one thing well |

---

## Task 1: Add replacePlaceholders Helper Function

**Files:**
- Create: `middleware.go` (after line 15, before `PrepareRequestMiddleware`)

**Step 1: Write the failing test**

Create test file `middleware_test.go` (add new test at end):

```go
func TestReplacePlaceholders(t *testing.T) {
    tests := []struct {
        name   string
        url    string
        params map[string]string
        want   string
    }{
        {
            name:   "single replacement",
            url:    "/users/{id}",
            params: map[string]string{"id": "123"},
            want:   "/users/123",
        },
        {
            name:   "multiple replacements",
            url:    "/users/{id}/posts/{postId}",
            params: map[string]string{"id": "123", "postId": "456"},
            want:   "/users/123/posts/456",
        },
        {
            name:   "no params - placeholder remains",
            url:    "/users/{id}",
            params: map[string]string{},
            want:   "/users/{id}",
        },
        {
            name:   "missing param - placeholder remains",
            url:    "/users/{id}/posts/{postId}",
            params: map[string]string{"id": "123"},
            want:   "/users/{id}/posts/{postId}",
        },
        {
            name:   "value with slash",
            url:    "/users/{id}/profile",
            params: map[string]string{"id": "user/name"},
            want:   "/users/user/name/profile",
        },
        {
            name:   "no placeholders",
            url:    "/users/123",
            params: map[string]string{"id": "456"},
            want:   "/users/123",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := replacePlaceholders(tt.url, tt.params)
            if got != tt.want {
                t.Errorf("replacePlaceholders() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestReplacePlaceholders ./`
Expected: FAIL with "undefined: replacePlaceholders"

**Step 3: Write minimal implementation**

Add to `middleware.go` after imports (around line 15):

```go
// replacePlaceholders replaces {key} with values from params.
func replacePlaceholders(url string, params map[string]string) string {
    for key, value := range params {
        placeholder := "{" + key + "}"
        url = strings.ReplaceAll(url, placeholder, value)
    }
    return url
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestReplacePlaceholders ./`
Expected: PASS

---

## Task 2: Update parseRequestURL to Use replacePlaceholders

**Files:**
- Modify: `middleware.go:40-94`

**Step 1: Write the failing test**

Update existing test in `middleware_test.go`. Modify these test cases in `Test_parseRequestURL`:

Change lines 23-35 (apply client path parameters):

```go
{
    name: "apply client path parameters",
    initClient: func(c *Client) {
        c.SetPathParams(map[string]string{
            "foo": "1",
            "bar": "2/3",  // note: no longer escaped
        })
    },
    initRequest: func(r *Request) {
        r.URL = "https://example.com/{foo}/{bar}"
    },
    expectedURL: "https://example.com/1/2/3",  // changed from %2F
},
```

Change lines 47-62 (apply request and client path parameters):

```go
{
    name: "apply request and client path parameters",
    initClient: func(c *Client) {
        c.SetPathParams(map[string]string{
            "foo": "1",  // will be ignored
            "bar": "2/3",
        })
    },
    initRequest: func(r *Request) {
        r.SetPathParams(map[string]string{
            "foo": "4/5",
        })
        r.URL = "https://example.com/{foo}/{bar}"
    },
    expectedURL: "https://example.com/4/5/2/3",  // changed from %2F
},
```

Delete test cases:
- Lines 64-102: all "raw path parameters" tests (4 test cases)
- Lines 117-125: "empty path parameter in URL"
- Lines 127-135: "not closed path parameter in URL"

**Step 2: Run test to verify it fails**

Run: `go test -v -run Test_parseRequestURL ./`
Expected: FAIL with URL mismatch due to escape differences

**Step 3: Replace parseRequestURL PathParams logic**

Replace lines 40-94 in `middleware.go` with:

```go
// Merge client and request path params
if len(c.PathParams())+len(r.PathParams) > 0 {
    // Merge client params (request params take precedence)
    for p, v := range c.PathParams() {
        if _, ok := r.PathParams[p]; !ok {
            r.PathParams[p] = v
        }
    }
    // Replace placeholders
    r.URL = replacePlaceholders(r.URL, r.PathParams)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run Test_parseRequestURL ./`
Expected: PASS

---

## Task 3: Remove unescapeQueryParams from Request

**Files:**
- Modify: `request.go:64`
- Modify: `request.go:415-420`
- Modify: `middleware.go:143-150`

**Step 1: Write the failing test**

No new test needed - removing feature means removing tests.

**Step 2: Remove unescapeQueryParams field**

Delete line 64 in `request.go`:

```go
unescapeQueryParams bool
```

**Step 3: Remove SetUnescapeQueryParams method**

Delete lines 415-420 in `request.go`:

```go
// SetUnescapeQueryParams sets whether to unescape query parameters.
// NOTE: Request failure is possible with non-standard usage.
func (r *Request) SetUnescapeQueryParams(unescape bool) *Request {
    r.unescapeQueryParams = unescape
    return r
}
```

**Step 4: Remove unescape logic from parseRequestURL**

Delete lines 143-150 in `middleware.go`:

```go
// GH#797 Unescape query parameters (non-standard - not recommended)
if r.unescapeQueryParams && len(reqURL.RawQuery) > 0 {
    // at this point, all errors caught up in the above operations
    // so ignore the return error on query unescape; I realized
    // while writing the unit test
    unescapedQuery, _ := url.QueryUnescape(reqURL.RawQuery)
    reqURL.RawQuery = strings.ReplaceAll(unescapedQuery, " ", "+") // otherwise request becomes bad request
}
```

**Step 5: Remove unescape test case**

Delete lines 296-312 in `middleware_test.go`:

```go
{
    name: "unescape query params",
    ...
},
```

**Step 6: Run tests to verify**

Run: `go test -v ./`
Expected: PASS (with fewer tests passing)

---

## Task 4: Simplify Request SetPathParam Methods

**Files:**
- Modify: `request.go:323-351`

**Step 1: Update SetPathParam (no escape)**

Replace lines 323-328 in `request.go`:

```go
// SetPathParam sets a single URL path parameter (replaces {key} in URL).
func (r *Request) SetPathParam(param, value string) *Request {
    r.PathParams[param] = value
    return r
}
```

**Step 2: Update SetPathParams (no escape)**

Replace lines 330-337 in `request.go`:

```go
// SetPathParams sets multiple URL path parameters.
func (r *Request) SetPathParams(params map[string]string) *Request {
    for p, v := range params {
        r.PathParams[p] = v
    }
    return r
}
```

**Step 3: Delete SetRawPathParam**

Delete lines 339-343 in `request.go`:

```go
// SetRawPathParam sets a URL path parameter without escaping.
func (r *Request) SetRawPathParam(param, value string) *Request {
    r.PathParams[param] = value
    return r
}
```

**Step 4: Delete SetRawPathParams**

Delete lines 345-351 in `request.go`:

```go
// SetRawPathParams sets multiple URL path parameters without escaping.
func (r *Request) SetRawPathParams(params map[string]string) *Request {
    for p, v := range params {
        r.SetRawPathParam(p, v)
    }
    return r
}
```

**Step 5: Run tests to verify**

Run: `go test -v -run TestGetPathParamAndPathParams ./`
Expected: PASS (behavior unchanged since using raw values)

---

## Task 5: Simplify Client SetPathParam Methods

**Files:**
- Modify: `client.go:723-755`

**Step 1: Update SetPathParam**

Replace lines 723-730 in `client.go`:

```go
// SetPathParam sets a single URL path parameter.
func (c *Client) SetPathParam(param, value string) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.pathParams[param] = value
    return c
}
```

**Step 2: Update SetPathParams**

Replace lines 732-739 in `client.go`:

```go
// SetPathParams sets multiple URL path parameters.
func (c *Client) SetPathParams(params map[string]string) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    for p, v := range params {
        c.pathParams[p] = v
    }
    return c
}
```

**Step 3: Delete SetRawPathParam**

Delete lines 741-747 in `client.go`:

```go
// SetRawPathParam sets a URL path parameter without escaping.
func (c *Client) SetRawPathParam(param, value string) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.pathParams[param] = value
    return c
}
```

**Step 4: Delete SetRawPathParams**

Delete lines 749-755 in `client.go`:

```go
// SetRawPathParams sets multiple URL path parameters without escaping.
func (c *Client) SetRawPathParams(params map[string]string) *Client {
    for p, v := range params {
        c.SetRawPathParam(p, v)
    }
    return c
}
```

**Step 5: Run tests to verify**

Run: `go test -v -run "Test.*Path" ./`
Expected: PASS

---

## Task 6: Remove Client unescapeQueryParams

**Files:**
- Modify: `client.go:111`
- Modify: `client.go:799-806`
- Modify: `client.go:294`

**Step 1: Remove field from struct**

Delete line 111 in `client.go`:

```go
unescapeQueryParams     bool
```

**Step 2: Remove SetUnescapeQueryParams method**

Delete lines 799-806 in `client.go`:

```go
// SetUnescapeQueryParams sets whether to unescape query parameters.
// NOTE: Request failure is possible with non-standard usage.
func (c *Client) SetUnescapeQueryParams(unescape bool) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.unescapeQueryParams = unescape
    return c
}
```

**Step 3: Remove field initialization**

Delete line 294 in `client.go`:

```go
unescapeQueryParams: c.unescapeQueryParams,
```

**Step 4: Run tests to verify**

Run: `go test -v ./`
Expected: PASS

---

## Task 7: Update Request_test.go Tests

**Files:**
- Modify: `request_test.go:1255-1274`

**Step 1: Update TestGetPathParamAndPathParams**

No changes needed - test uses `@` in email which is now preserved as-is.

**Step 2: Verify test still passes**

Run: `go test -v -run TestGetPathParamAndPathParams ./`
Expected: PASS

---

## Task 8: Update Benchmark Test

**Files:**
- Modify: `benchmark_test.go:9-22`

**Step 1: Update benchmark to use new API**

Replace lines 9-22 in `benchmark_test.go`:

```go
func Benchmark_parseRequestURL_PathParams(b *testing.B) {
    c := New().SetPathParams(map[string]string{
        "foo": "1",
        "bar": "2/3",
    })
    r := c.R().SetPathParams(map[string]string{
        "foo": "4/5",
    })
    r.URL = "https://example.com/{foo}/{bar}"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = parseRequestURL(c, r)
    }
}
```

**Step 2: Run benchmark**

Run: `go test -bench=Benchmark_parseRequestURL_PathParams -benchmem ./`
Expected: Benchmark runs successfully

---

## Task 9: Run All Tests

**Step 1: Run full test suite**

Run: `go test -v ./`
Expected: All tests PASS

**Step 2: Run with race detector**

Run: `go test -race ./`
Expected: No race conditions

**Step 3: Run benchmarks**

Run: `go test -bench=. -benchmem ./`
Expected: All benchmarks complete

---

## Task 10: Build Verification

**Step 1: Build the package**

Run: `go build ./`
Expected: Binary builds successfully

**Step 2: Verify no compilation errors**

Run: `go vet ./`
Expected: No warnings

---

## Summary

**Files Modified:**
1. `middleware.go` - Add replacePlaceholders, simplify parseRequestURL, remove unescape
2. `request.go` - Simplify SetPath* methods, remove unescape
3. `client.go` - Simplify SetPath* methods, remove unescape
4. `middleware_test.go` - Update/remove test cases
5. `benchmark_test.go` - Update benchmark
6. `request_test.go` - Verify existing tests

**Lines Removed:** ~60-70 lines
**Lines Added:** ~30 lines
**Net Change:** ~30-40 lines removed

**Breaking Changes:**
- `SetPathParam` no longer escapes - user must handle if needed
- `SetRawPathParam` removed - use `SetPathParam`
- `SetUnescapeQueryParams` removed - feature deleted
