# Implementation Plan: Remove Curl Functionality

**Design:** [design.md](./design.md)
**Created:** 2025-12-25
**Estimated Tasks:** 10

## Overview

This plan breaks down the removal of curl functionality into 10 bite-sized tasks following the design specification.

## Tasks

### Task 1: Delete curl.go
**File:** `curl.go`
**Action:** Delete entire file (~100 lines)

**Files to delete:**
- `curl.go`

**Verification:**
```bash
ls curl.go 2>&1 | grep -q "No such file"
```

---

### Task 2: Delete curl_test.go
**File:** `curl_test.go`
**Action:** Delete entire file (~270 lines)

**Files to delete:**
- `curl_test.go`

**Verification:**
```bash
ls curl_test.go 2>&1 | grep -q "No such file"
```

---

### Task 3: Remove curl fields from Client struct
**File:** `client.go`
**Lines:** 127-128

**Remove these lines:**
```go
generateCurlCmd         bool
debugLogCurlCmd         bool
```

**Verification:**
```bash
grep -n "generateCurlCmd\|debugLogCurlCmd" client.go | grep -v "^//"
# Should only match in requestClientValues struct
```

---

### Task 4: Remove curl methods from Client
**File:** `client.go`
**Lines:** 1248-1301

**Remove these methods:**
- `EnableGenerateCurlCmd()` (lines 1260-1263)
- `DisableGenerateCurlCmd()` (lines 1266-1270)
- `SetGenerateCurlCmd()` (lines 1286-1291)
- `SetDebugLogCurlCmd()` (lines 1296-1301)
- Associated comments (lines 1248-1258, 1265-1285, 1293-1295)

**Exact block to remove:**
```go
// EnableGenerateCurlCmd method enables the generation of curl command at the
// client instance level.
//
// By default, Resty does not log the curl command in the debug log since it has the potential
// to leak sensitive data unless explicitly enabled via [Client.SetDebugLogCurlCmd] or
// [Request.SetDebugLogCurlCmd].
//
// NOTE: Use with care.
//   - Potential to leak sensitive data from [Request] and [Response] in the debug log
//   - Additional memory usage since the request body was reread.
//   - curl body is not generated for [io.Reader] and multipart request flow.
func (c *Client) EnableGenerateCurlCmd() *Client {
	c.SetGenerateCurlCmd(true)
	return c
}

// DisableGenerateCurlCmd method disables the option set by [Client.EnableGenerateCurlCmd] or
// [Client.SetGenerateCurlCmd].
func (c *Client) DisableGenerateCurlCmd() *Client {
	c.SetGenerateCurlCmd(false)
	return c
}

// SetGenerateCurlCmd method is used to turn on/off the generate curl command at the
// client instance level.
//
// By default, Resty does not log the curl command in the debug log since it has the potential
// to leak sensitive data unless explicitly enabled via [Client.SetDebugLogCurlCmd] or
// [Request.SetDebugLogCurlCmd].
//
// NOTE: Use with care.
//   - Potential to leak sensitive data from [Request] and [Response] in the debug log
//   - Additional memory usage since the request body was reread.
//   - curl body is not generated for [io.Reader] and multipart request flow.
//
// It can be overridden at the request level; see [Request.SetGenerateCurlCmd]
func (c *Client) SetGenerateCurlCmd(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.generateCurlCmd = b
	return c
}

// SetDebugLogCurlCmd method enables the curl command to be logged in the debug log.
//
// It can be overridden at the request level; see [Request.SetDebugLogCurlCmd]
func (c *Client) SetDebugLogCurlCmd(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debugLogCurlCmd = b
	return c
}
```

**Verification:**
```bash
grep -n "EnableGenerateCurlCmd\|DisableGenerateCurlCmd\|SetGenerateCurlCmd\|SetDebugLogCurlCmd" client.go
# Should have no results
```

---

### Task 5: Remove curl field propagation in requestClientValues
**File:** `client.go`
**Lines:** 412-413

**Remove from struct initialization:**
```go
generateCurlCmd:     c.generateCurlCmd,
debugLogCurlCmd:     c.debugLogCurlCmd,
```

**After removal, the struct should be:**
```go
return &Request{
	client:              c,
	baseURL:             c.baseURL,
	multipartFields:     make([]*MultipartField, 0),
	jsonEscapeHTML:      c.jsonEscapeHTML,
	log:                 c.log,
	setContentLength:    c.setContentLength,
	unescapeQueryParams: c.unescapeQueryParams,
}
```

**Verification:**
```bash
# Check that requestClientValues no longer has these fields
grep -A10 "requestClientValues struct" client.go | grep -E "generateCurlCmd|debugLogCurlCmd"
# Should have no results
```

---

### Task 6: Remove curl fields from Request struct
**File:** `request.go`
**Lines:** 71-73

**Remove these lines:**
```go
resultCurlCmd       string
generateCurlCmd     bool
debugLogCurlCmd     bool
```

**Verification:**
```bash
grep -n "resultCurlCmd\|generateCurlCmd\|debugLogCurlCmd" request.go
# Should have no results
```

---

### Task 7: Remove curl methods from Request
**File:** `request.go`
**Lines:** 842-916

**Remove these methods:**
- `EnableGenerateCurlCmd()` (lines 855-858)
- `DisableGenerateCurlCmd()` (lines 864-867)
- `SetGenerateCurlCmd()` (lines 882-885)
- `SetDebugLogCurlCmd()` (lines 891-894)
- `CurlCmd()` (lines 897-899)
- `generateCurlCommand()` (lines 901-916)
- Associated comments (lines 842-854, 860-881, 887-896)

**Exact block to remove:**
```go
// EnableGenerateCurlCmd method enables the generation of curl commands for the current request.
//
// By default, Resty does not log the curl command in the debug log since it has the potential
// to leak sensitive data unless explicitly enabled via [Request.SetDebugLogCurlCmd] or
// [Client.SetDebugLogCurlCmd].
//
// It overrides the options set in the [Client].
//
// NOTE: Use with care.
//   - Potential to leak sensitive data from [Request] and [Response] in the debug log
//   - Additional memory usage since the request body was reread.
//   - curl body is not generated for [io.Reader] and multipart request flow.
func (r *Request) EnableGenerateCurlCmd() *Request {
	r.SetGenerateCurlCmd(true)
	return r
}

// DisableGenerateCurlCmd method disables the option set by [Request.EnableGenerateCurlCmd] or
// [Request.SetGenerateCurlCmd].
//
// It overrides the options set in the [Client].
func (r *Request) DisableGenerateCurlCmd() *Request {
	r.SetGenerateCurlCmd(false)
	return r
}

// SetGenerateCurlCmd method is used to turn on/off the generate curl command for the current request.
//
// By default, Resty does not log the curl command in the debug log since it has the potential
// to leak sensitive data unless explicitly enabled via [Request.SetDebugLogCurlCmd] or
// [Client.SetDebugLogCurlCmd].
//
// It overrides the options set by the [Client.SetGenerateCurlCmd]
//
// NOTE: Use with care.
//   - Potential to leak sensitive data from [Request] and [Response] in the debug log
//   - Additional memory usage since the request body was reread.
//   - curl body is not generated for [io.Reader] and multipart request flow.
func (r *Request) SetGenerateCurlCmd(b bool) *Request {
	r.generateCurlCmd = b
	return r
}

// SetDebugLogCurlCmd method enables the curl command to be logged in the debug log
// for the current request.
//
// It can be overridden at the request level; see [Client.SetDebugLogCurlCmd]
func (r *Request) SetDebugLogCurlCmd(b bool) *Request {
	r.debugLogCurlCmd = b
	return r
}

// CurlCmd method generates the curl command for the request.
func (r *Request) CurlCmd() string {
	return r.generateCurlCommand()
}

func (r *Request) generateCurlCommand() string {
	if !r.generateCurlCmd {
		return ""
	}
	if len(r.resultCurlCmd) > 0 {
		return r.resultCurlCmd
	}
	if r.RawRequest == nil {
		if len(r.resultCurlCmd) == 0 {
			r.resultCurlCmd = buildCurlCmd(r)
		}
	} else {
		r.resultCurlCmd = buildCurlCmd(r)
	}
	return r.resultCurlCmd
}
```

**Verification:**
```bash
grep -n "EnableGenerateCurlCmd\|DisableGenerateCurlCmd\|SetGenerateCurlCmd\|SetDebugLogCurlCmd\|CurlCmd" request.go
# Should have no results
```

---

### Task 8: Remove CurlCmd from RequestDebugLog struct
**File:** `debug.go`
**Line:** 38

**Remove from struct:**
```go
CurlCmd string `json:"curl_cmd"`
```

**After removal, the struct should be:**
```go
type RequestDebugLog struct {
	Header  http.Header `json:"header"`
	Body    string      `json:"body"`
}
```

**Verification:**
```bash
grep -n "CurlCmd" debug.go
# Should only match in the code block we'll remove in Task 9
```

---

### Task 9: Remove curl logging from debug.go
**File:** `debug.go`

**Remove these two code blocks:**

**Block 1 (lines 63-66):**
```go
if len(req.CurlCmd) > 0 {
	debugLog += "~~~ REQUEST(CURL) ~~~\n" +
		fmt.Sprintf("	%v\n", req.CurlCmd)
}
```

**Block 2 (lines 164-166):**
```go
if r.generateCurlCmd && r.debugLogCurlCmd {
	rdl.CurlCmd = r.resultCurlCmd
}
```

**Verification:**
```bash
grep -n "CurlCmd\|generateCurlCmd\|debugLogCurlCmd" debug.go
# Should have no results
```

---

### Task 10: Remove EnableGenerateCurlCmd from stream_test.go
**File:** `stream_test.go`
**Line:** 30

**Change:**
```go
// Before:
client := New().EnableGenerateCurlCmd()

// After:
client := New()
```

**Verification:**
```bash
grep -n "EnableGenerateCurlCmd" stream_test.go
# Should have no results
```

---

## Final Verification

After all tasks complete:

```bash
# Build check
go build ./...

# Test check
go test ./... -v

# Static analysis
go vet ./...

# Dependency check - should find nothing
grep -r "EnableGenerateCurlCmd\|DisableGenerateCurlCmd\|SetGenerateCurlCmd\|SetDebugLogCurlCmd\|CurlCmd" . --include="*.go" | grep -v "\.md"

# Check deleted files
ls curl.go curl_test.go 2>&1 | grep -q "No such file"
```

## Summary

- **Tasks:** 10
- **Files modified:** 4 files
- **Files deleted:** 2 files
- **Lines removed:** ~150-200
- **API methods removed:** 10 public methods
