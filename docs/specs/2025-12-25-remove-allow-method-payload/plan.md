# Remove AllowMethodGetPayload/AllowMethodDeletePayload Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** Remove the AllowMethodGetPayload and AllowMethodDeletePayload functionality to simplify the API and let users control payload behavior themselves.

**Architecture:** Remove conditional checks from isPayloadSupported() and delete all related fields, methods, and tests. This allows GET/DELETE requests to have body without library-imposed restrictions.

**Tech Stack:** Go 1.23+, standard library only

---

## Task 1: Simplify isPayloadSupported() in request.go

**Files:**
- Modify: `request.go:1330-1349`

**Step 1: Modify isPayloadSupported() to remove conditional checks**

Remove the AllowMethodGetPayload and AllowMethodDeletePayload conditions. The method should simply allow GET, DELETE, POST, PUT, and PATCH methods to have payload.

Before:
```go
func (r *Request) isPayloadSupported() bool {
	if r.Method == "" {
		r.Method = MethodGet
	}

	if r.Method == MethodGet && r.AllowMethodGetPayload {
		return true
	}

	// More info, refer to GH#881
	if r.Method == MethodDelete && r.AllowMethodDeletePayload {
		return true
	}

	if r.Method == MethodPost || r.Method == MethodPut || r.Method == MethodPatch {
		return true
	}

	return false
}
```

After:
```go
func (r *Request) isPayloadSupported() bool {
	if r.Method == "" {
		r.Method = MethodGet
	}

	return r.Method == MethodGet ||
		r.Method == MethodDelete ||
		r.Method == MethodPost ||
		r.Method == MethodPut ||
		r.Method == MethodPatch
}
```

**Step 2: Run tests to verify changes**

Run: `go test -v -run TestRequest_isPayloadSupported`

Expected: Tests should still pass (isPayloadSupported tests now allow GET/DELETE without requiring flags)

---

## Task 2: Remove Request struct fields

**Files:**
- Modify: `request.go:54-55`

**Step 1: Remove fields from Request struct**

Remove these two lines from the Request struct definition:
```go
	AllowMethodGetPayload      bool
	AllowMethodDeletePayload   bool
```

**Step 2: Run tests to verify compilation**

Run: `go build ./...`

Expected: Compilation errors about undefined SetAllowMethodGetPayload and SetAllowMethodDeletePayload methods (we will fix these in next tasks)

---

## Task 3: Remove Request setter methods

**Files:**
- Modify: `request.go:931-953`

**Step 1: Remove SetAllowMethodGetPayload and SetAllowMethodDeletePayload methods**

Remove the entire methods including comments:
```go
// SetAllowMethodGetPayload method allows the GET method with payload on the request level.
// By default, Resty does not allow.
//
//	client.R().SetAllowMethodGetPayload(true)
//
// It overrides the option set by the [Client.SetAllowMethodGetPayload]
func (r *Request) SetAllowMethodGetPayload(allow bool) *Request {
	r.AllowMethodGetPayload = allow
	return r
}

// SetAllowMethodDeletePayload method allows the DELETE method with payload on the request level.
// By default, Resty does not allow.
//
//	client.R().SetAllowMethodDeletePayload(true)
//
// More info, refer to GH#881
//
// It overrides the option set by the [Client.SetAllowMethodDeletePayload]
func (r *Request) SetAllowMethodDeletePayload(allow bool) *Request {
	r.AllowMethodDeletePayload = allow
	return r
}
```

**Step 2: Run tests to verify compilation**

Run: `go build ./...`

Expected: Fewer compilation errors, now only about client.go methods

---

## Task 4: Remove Client struct fields

**Files:**
- Modify: `client.go:164-165`

**Step 1: Remove fields from Client struct**

Remove these two lines from the Client struct definition:
```go
	allowMethodGetPayload    bool
	allowMethodDeletePayload bool
```

**Step 2: Run tests to verify compilation**

Run: `go build ./...`

Expected: Compilation errors about undefined Client methods (we fix in next task)

---

## Task 5: Remove Client getter/setter methods

**Files:**
- Modify: `client.go:868-912`

**Step 1: Remove all four methods**

Remove these methods including comments:
```go
// AllowMethodGetPayload method returns `true` if the client is enabled to allow
// payload with GET method; otherwise, it is `false`.
func (c *Client) AllowMethodGetPayload() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.allowMethodGetPayload
}

// SetAllowMethodGetPayload method allows the GET method with payload on the Resty client.
// By default, Resty does not allow.
//
//	client.SetAllowMethodGetPayload(true)
//
// It can be overridden at the request level. See [Request.SetAllowMethodGetPayload]
func (c *Client) SetAllowMethodGetPayload(allow bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.allowMethodGetPayload = allow
	return c
}

// AllowMethodDeletePayload method returns `true` if the client is enabled to allow
// payload with DELETE method; otherwise, it is `false`.
//
// More info, refer to GH#881
func (c *Client) AllowMethodDeletePayload() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.allowMethodDeletePayload
}

// SetAllowMethodDeletePayload method allows the DELETE method with payload on the Resty client.
// By default, Resty does not allow.
//
//	client.SetAllowMethodDeletePayload(true)
//
// More info, refer to GH#881
//
// It can be overridden at the request level. See [Request.SetAllowMethodDeletePayload]
func (c *Client) SetAllowMethodDeletePayload(allow bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.allowMethodDeletePayload = allow
	return c
}
```

**Step 2: Run tests to verify compilation**

Run: `go build ./...`

Expected: Compilation errors about copying removed fields in Request initialization (next task)

---

## Task 6: Remove field copying in Client.NewRequest()

**Files:**
- Modify: `client.go:461-462`

**Step 1: Remove field assignments**

Remove these two lines from the Request initialization:
```go
	AllowMethodGetPayload:      c.allowMethodGetPayload,
	AllowMethodDeletePayload:   c.allowMethodDeletePayload,
```

**Step 2: Run tests to verify compilation**

Run: `go build ./...`

Expected: Clean build, no compilation errors

---

## Task 7: Remove TestClientAllowMethodGetPayload test

**Files:**
- Modify: `client_test.go:337-380`

**Step 1: Remove entire test function**

Delete the complete `TestClientAllowMethodGetPayload` function and all its sub-tests.

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientAllowMethodGetPayload`

Expected: No tests found (function removed)

---

## Task 8: Remove TestClientAllowMethodDeletePayload test

**Files:**
- Modify: `client_test.go:382-429`

**Step 1: Remove entire test function**

Delete the complete `TestClientAllowMethodDeletePayload` function and all its sub-tests.

**Step 2: Run tests to verify**

Run: `go test -v -run TestClientAllowMethodDeletePayload`

Expected: No tests found (function removed)

---

## Task 9: Remove middleware test cases

**Files:**
- Modify: `middleware_test.go:492-513` (GET cases)
- Modify: `middleware_test.go:564-573` (DELETE case)

**Step 1: Remove GET test cases**

Remove these two test cases from the table:
```go
{
	name: "string body with GET method and AllowMethodGetPayload by client",
	initClient: func(c *Client) {
		c.SetAllowMethodGetPayload(true)
	},
	initRequest: func(r *Request) {
		r.SetBody("foo")
		r.Method = http.MethodGet
	},
	expectedBodyBuf:     []byte("foo"),
	expectedContentType: plainTextType,
},
{
	name: "string body with GET method and AllowMethodGetPayload by request",
	initRequest: func(r *Request) {
		r.SetAllowMethodGetPayload(true)
		r.SetBody("foo")
		r.Method = http.MethodGet
	},
	expectedBodyBuf:     []byte("foo"),
	expectedContentType: plainTextType,
},
```

**Step 2: Remove DELETE test case**

Remove this test case:
```go
{
	name: "string body with DELETE method with AllowMethodDeletePayload by request",
	initRequest: func(r *Request) {
		r.SetAllowMethodDeletePayload(true)
		r.SetBody("foo")
		r.Method = http.MethodDelete
	},
	expectedBodyBuf:     []byte("foo"),
	expectedContentType: plainTextType,
},
```

**Step 3: Run tests to verify**

Run: `go test -v -run TestRequest_buildHTTPRequest`

Expected: Tests pass (GET/DELETE with body should now work without flags)

---

## Task 10: Remove request_test.go test cases

**Files:**
- Modify: `request_test.go:1920-1942` (GET tests)
- Modify: `request_test.go:1964-1976` (DELETE tests)

**Step 1: Find and remove isPayloadSupported GET test cases**

Look for tests that call `SetAllowMethodGetPayload(true)` and remove them. Based on grep results:
- Test case around line 1927
- Test case around line 1939

**Step 2: Find and remove isPayloadSupported DELETE test case**

Look for tests that call `SetAllowMethodDeletePayload(true)` and remove them. Based on grep results:
- Test case around line 1972

**Step 3: Run tests to verify**

Run: `go test -v -run TestRequest_isPayloadSupported`

Expected: Tests pass (or tests removed if they only tested the removed functionality)

---

## Task 11: Clean up stream_test.go

**Files:**
- Modify: `stream_test.go:36`

**Step 1: Remove SetAllowMethodGetPayload call**

Find and remove `.SetAllowMethodGetPayload(true)` from the test:

Before:
```go
	resp, err := client.R().SetBody("{}").
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetForceResponseContentType("application/json").
		SetAllowMethodGetPayload(true).
		SetResponseBodyUnlimitedReads(true).
		SetResult(&x).
		Get(server.URL + "/test")
```

After:
```go
	resp, err := client.R().SetBody("{}").
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetForceResponseContentType("application/json").
		SetResponseBodyUnlimitedReads(true).
		SetResult(&x).
		Get(server.URL + "/test")
```

**Step 2: Run tests to verify**

Run: `go test -v -run TestStreamNilJSON`

Expected: Test passes (GET with body works without flag)

---

## Task 12: Final verification

**Files:**
- None (verification only)

**Step 1: Run full test suite**

Run: `go test ./... -v`

Expected: All tests pass

**Step 2: Verify build**

Run: `go build ./...`

Expected: Clean build, no errors or warnings

**Step 3: Verify no remaining references**

Run: `grep -r "AllowMethodGetPayload\|AllowMethodDeletePayload" --include="*.go" .`

Expected: No results (all references removed)

**Step 4: Check for unused imports or variables**

Run: `go vet ./...`

Expected: Clean output, no warnings

---

## Summary

After completing all tasks:
- ~150-200 lines of code removed
- GET/DELETE requests can have body without library restrictions
- API is simpler with fewer concepts
- Breaking change for users calling removed methods, but functionality still works
