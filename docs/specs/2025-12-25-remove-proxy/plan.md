# Remove Proxy and HTTPTransport Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use @evo-executing-plans to implement this plan task-by-task.

**Goal:** Remove internal Proxy management logic and HTTPTransport() type assertion method

**Architecture:** Delete proxyURL field, 5 related methods, and 1 error variable from Client struct. Modify SetTransport to panic on nil. Users configure proxy directly via http.Transport.

**Tech Stack:** Go 1.23+, net/http standard library

---

## Task 1: Write failing test for SetTransport nil panic

**Files:**
- Modify: `client_test.go`

**Step 1: Write the failing test**

Append to `client_test.go` (after existing tests):

```go
func TestClientSetTransport_NilPanic(t *testing.T) {
	c := New()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("SetTransport(nil) should panic")
		}
	}()
	c.SetTransport(nil)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestClientSetTransport_NilPanic`

Expected: FAIL (currently SetTransport(nil) does nothing, no panic)

---

## Task 2: Remove proxyURL field from Client struct

**Files:**
- Modify: `client.go:110`

**Step 1: Remove the field**

In `client.go` Client struct (around line 110), remove:
```go
proxyURL *url.URL
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (compilation errors in code using proxyURL)

---

## Task 3: Remove ProxyURL method

**Files:**
- Modify: `client.go:650-655`

**Step 1: Delete the method**

Remove entirely:
```go
// ProxyURL returns the proxy URL.
func (c *Client) ProxyURL() *url.URL {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.proxyURL
}
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (if any tests call ProxyURL)

---

## Task 4: Remove SetProxy method

**Files:**
- Modify: `client.go:657-676`

**Step 1: Delete the method**

Remove entirely:
```go
// SetProxy sets the proxy URL.
func (c *Client) SetProxy(proxyURL string) *Client {
	transport, err := c.HTTPTransport()
	if err != nil {
		c.Logger().Errorf("%v", err)
		return c
	}

	pURL, err := url.Parse(proxyURL)
	if err != nil {
		c.Logger().Errorf("%v", err)
		return c
	}

	c.lock.Lock()
	c.proxyURL = pURL
	transport.Proxy = http.ProxyURL(c.proxyURL)
	c.lock.Unlock()
	return c
}
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (if any tests call SetProxy)

---

## Task 5: Remove RemoveProxy method

**Files:**
- Modify: `client.go:678-691`

**Step 1: Delete the method**

Remove entirely:
```go
// RemoveProxy removes the proxy configuration.
func (c *Client) RemoveProxy() *Client {
	transport, err := c.HTTPTransport()
	if err != nil {
		c.Logger().Errorf("%v", err)
		return c
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.proxyURL = nil
	transport.Proxy = nil
	return c
}
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (if any tests call RemoveProxy)

---

## Task 6: Remove HTTPTransport method

**Files:**
- Modify: `client.go:708-716`

**Step 1: Delete the method**

Remove entirely:
```go
// HTTPTransport returns the http.Transport.
func (c *Client) HTTPTransport() (*http.Transport, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		return transport, nil
	}
	return nil, ErrNotHttpTransportType
}
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (if any tests call HTTPTransport)

---

## Task 7: Remove IsProxySet method

**Files:**
- Modify: `client.go:878-881`

**Step 1: Delete the method**

Remove entirely:
```go
// IsProxySet returns whether proxy is configured.
func (c *Client) IsProxySet() bool {
	return c.ProxyURL() != nil
}
```

**Step 2: Run tests to verify breakage**

Run: `go test ./...`

Expected: FAIL (if any tests call IsProxySet)

---

## Task 8: Modify SetTransport to panic on nil

**Files:**
- Modify: `client.go:725-734`

**Step 1: Replace the method**

Replace existing SetTransport with:
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

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestClientSetTransport_NilPanic`

Expected: PASS

---

## Task 9: Remove proxyURL cloning from Clone method

**Files:**
- Modify: `client.go:908-910`

**Step 1: Remove the cloning logic**

In Clone method, remove these lines:
```go
if c.proxyURL != nil {
	cc.proxyURL, _ = url.Parse(c.proxyURL.String())
}
```

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientClone`

Expected: PASS

---

## Task 10: Remove ErrNotHttpTransportType error variable

**Files:**
- Modify: `client.go:45`

**Step 1: Remove the variable**

In var block, remove:
```go
ErrNotHttpTransportType = errors.New("resty: not a http.Transport type")
```

**Step 2: Run tests to verify**

Run: `go test ./...`

Expected: PASS (if no code uses this error)

---

## Task 11: Remove TestClientProxy test

**Files:**
- Modify: `client_test.go:107-126`

**Step 1: Delete the test function**

Remove entirely:
```go
func TestClientProxy(t *testing.T) {
	ts := createGetServer(t)
	defer ts.Close()

	c := dcnl()
	c.SetTimeout(1 * time.Second)
	c.SetProxy("http://sampleproxy:8888")

	resp, err := c.R().Get(ts.URL)
	assertNotNil(t, resp)
	assertNotNil(t, err)

	// error
	c.SetProxy("//not.a.user@%66%6f%6f.com:8888")

	resp, err = c.R().
		Get(ts.URL)
	assertNotNil(t, err)
	assertNotNil(t, resp)
}
```

**Step 2: Run tests to verify**

Run: `go test -v ./...`

Expected: PASS

---

## Task 12: Modify TestClientSetTransport test

**Files:**
- Modify: `client_test.go:170-186`

**Step 1: Update test to remove HTTPTransport call**

Replace test with:
```go
func TestClientSetTransport(t *testing.T) {
	ts := createGetServer(t)
	defer ts.Close()
	client := dcnl()

	transport := &http.Transport{
		// something like Proxying to httptest.Server, etc...
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(ts.URL)
		},
	}
	client.SetTransport(transport)

	// Verify transport was set via getter
	assertEqual(t, true, client.Transport() == transport)
}
```

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientSetTransport`

Expected: PASS

---

## Task 13: Modify TestClientSettingsCoverage test

**Files:**
- Modify: `client_test.go:233-244`

**Step 1: Remove proxy and HTTPTransport test code**

In TestClientSettingsCoverage, remove the Custom Transport scenario section (lines 233-244):
```go
// [Start] Custom Transport scenario
ct := dcnl()
ct.SetTransport(&CustomRoundTripper1{})
_, err := ct.HTTPTransport()
assertNotNil(t, err)
assertEqual(t, ErrNotHttpTransportType, err)

ct.SetProxy("http://localhost:8080")
ct.RemoveProxy()

ct.outputLogTo(io.Discard)
// [End] Custom Transport scenario
```

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientSettingsCoverage`

Expected: PASS

---

## Task 14: Modify TestClientClone test

**Files:**
- Modify: `client_test.go:847`

**Step 1: Remove SetProxy call**

Remove line:
```go
parent.SetProxy("http://localhost:8080")
```

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientClone`

Expected: PASS

---

## Task 15: Run full test suite

**Step 1: Run all tests**

Run: `go test -v ./...`

Expected: PASS (all tests pass)

**Step 2: Build to verify no compilation errors**

Run: `go build ./...`

Expected: SUCCESS (no errors)

---

## Task 16: Verify design compliance

**Step 1: Check against docs/constitution.md**

Verify:
- 1.1 YAGNI: Removed unnecessary proxy abstraction
- 1.2 Standard Library First: Users use http.Transport directly
- 2.1 TDD: Tests written before implementation
- 3.1 Error Handling: Panic on invalid SetTransport input

**Step 2: Verify code reduction**

Run: `git diff --stat`

Expected: Net reduction of ~50-70 lines

---

## Summary

**Tasks:** 16
**Estimated lines removed:** ~50-70
**Breaking changes:** SetProxy, RemoveProxy, ProxyURL, IsProxySet, HTTPTransport removed
