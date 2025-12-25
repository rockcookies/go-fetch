# Remove Client.Close() and OnCloseHook Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** Remove the `Client.Close()` method and `OnCloseHook` callback functionality (~50 lines)

**Architecture:** Delete type definition, struct field, public/private methods, and tests in order to avoid compilation errors

**Tech Stack:** Go 1.23+, standard testing

---

## Task 1: Delete CloseHook Type Definition

**Files:**
- Modify: `client.go:76-77`

**Step 1: Delete the type definition**

Delete lines 76-77:
```go
// DELETE these lines:
// CloseHook is a callback for client close.
CloseHook func()
```

**Step 2: Verify compilation**

Run: `go build .`
Expected: FAIL - `CloseHook` is referenced in `closeHooks` field

---

## Task 2: Delete closeHooks Field from Client Struct

**Files:**
- Modify: `client.go:116`

**Step 1: Delete the struct field**

Delete line 116 from Client struct:
```go
// DELETE this line:
closeHooks              []CloseHook
```

**Step 2: Verify compilation**

Run: `go build .`
Expected: FAIL - `closeHooks` is referenced in `OnClose()` method

---

## Task 3: Delete OnClose() Method

**Files:**
- Modify: `client.go:382-388`

**Step 1: Delete the OnClose() method**

Delete lines 382-388:
```go
// DELETE these lines:
// OnClose adds a callback for client close.
func (c *Client) OnClose(h CloseHook) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.closeHooks = append(c.closeHooks, h)
    return c
}
```

**Step 2: Verify compilation**

Run: `go build .`
Expected: FAIL - `closeHooks` is referenced in `Close()` method

---

## Task 4: Delete Close() Method

**Files:**
- Modify: `client.go:817-823`

**Step 1: Delete the Close() method**

Delete lines 817-823:
```go
// DELETE these lines:
// Close performs cleanup activities.
func (c *Client) Close() error {
    // Execute close hooks first
    c.onCloseHooks()

    return nil
}
```

**Step 2: Verify compilation**

Run: `go build .`
Expected: FAIL - `onCloseHooks()` is referenced in `Close()` but `onCloseHooks()` method still exists

---

## Task 5: Delete onCloseHooks() Private Helper

**Files:**
- Modify: `client.go:951-958`

**Step 1: Delete the onCloseHooks() method**

Delete lines 951-958:
```go
// DELETE these lines:
// Helper to run closeHooks hooks.
func (c *Client) onCloseHooks() {
    c.lock.RLock()
    defer c.lock.RUnlock()
    for _, h := range c.closeHooks {
        h()
    }
}
```

**Step 2: Verify compilation**

Run: `go build .`
Expected: PASS - all client.go deletions complete

---

## Task 6: Delete TestClientOnClose

**Files:**
- Modify: `client_test.go:938-949`

**Step 1: Delete the test function**

Delete lines 938-949:
```go
// DELETE these lines:
func TestClientOnClose(t *testing.T) {
    var hookExecuted bool

    c := dcnl()
    c.OnClose(func() {
        hookExecuted = true
    })

    err := c.Close()
    assertNil(t, err)
    assertEqual(t, true, hookExecuted)
}
```

**Step 2: Verify tests compile**

Run: `go test -c .`
Expected: PASS - compiles without this test

---

## Task 7: Delete TestClientOnCloseMultipleHooks

**Files:**
- Modify: `client_test.go:951-968`

**Step 1: Delete the test function**

Delete lines 951-968:
```go
// DELETE these lines:
func TestClientOnCloseMultipleHooks(t *testing.T) {
    var executionOrder []string

    c := dcnl()
    c.OnClose(func() {
        executionOrder = append(executionOrder, "first")
    })
    c.OnClose(func() {
        executionOrder = append(executionOrder, "second")
    })
    c.OnClose(func() {
        executionOrder = append(executionOrder, "third")
    })

    err := c.Close()
    assertNil(t, err)
    assertEqual(t, []string{"first", "second", "third"}, executionOrder)
}
```

**Step 2: Verify tests compile**

Run: `go test -c .`
Expected: PASS - compiles without this test

---

## Task 8: Verify No Residual References

**Files:**
- None (verification only)

**Step 1: Search for CloseHook references**

Run: `grep -r "CloseHook" --include="*.go" .`
Expected: No results

**Step 2: Search for closeHooks references**

Run: `grep -r "closeHooks" --include="*.go" .`
Expected: No results

**Step 3: Search for OnClose references**

Run: `grep -r "OnClose" --include="*.go" . | grep -v "// OnClose"`
Expected: No results (only comments)

**Step 4: Search for Client.Close references**

Run: `grep -r "\.Close()" --include="*.go" . | grep -v "test" | grep -v "resp\." | grep -v "defer ts\." | grep -v "defer file\." | grep -v "_ = f\."`
Expected: No `c.Close()` or `client.Close()` patterns

---

## Task 9: Run Full Test Suite

**Files:**
- None (verification only)

**Step 1: Run all tests**

Run: `go test -v ./...`
Expected: All tests pass

**Step 2: Build final binary**

Run: `go build .`
Expected: Success

---

## Summary

**Total Tasks:** 9
**Lines Removed:** ~56
**Files Modified:** 2 (`client.go`, `client_test.go`)

**Deleted Items:**
- `CloseHook` type (2 lines)
- `closeHooks` struct field (1 line)
- `OnClose()` method (7 lines)
- `Close()` method (7 lines)
- `onCloseHooks()` method (8 lines)
- `TestClientOnClose` (12 lines)
- `TestClientOnCloseMultipleHooks` (18 lines)
