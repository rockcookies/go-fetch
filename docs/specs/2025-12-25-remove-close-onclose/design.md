# Design: Remove Client.Close() and OnCloseHook

**Date:** 2025-12-25
**Status:** Proposed
**Impact:** ~50 lines removed

## Overview

Remove the `Client.Close()` method and `OnCloseHook` callback functionality to simplify the client lifecycle management. Users will manage their own resource cleanup using standard Go patterns (e.g., `defer`).

## Rationale

The current `Close()` method only executes `closeHooks` and performs no actual cleanup. This adds unnecessary complexity for a feature that provides little value. Standard Go patterns like `defer` are sufficient for resource management.

## Changes

### 1. Remove Type Definition (client.go:76-77)

```go
// DELETE:
// CloseHook is a callback for client close.
CloseHook func()
```

### 2. Remove Client Struct Field (client.go:116)

```go
// DELETE from Client struct:
closeHooks []CloseHook
```

### 3. Remove Public Methods

**OnClose()** (client.go:382-388):
```go
// DELETE:
func (c *Client) OnClose(h CloseHook) *Client {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.closeHooks = append(c.closeHooks, h)
    return c
}
```

**Close()** (client.go:817-823):
```go
// DELETE:
func (c *Client) Close() error {
    c.onCloseHooks()
    return nil
}
```

### 4. Remove Private Helper (client.go:951-958)

```go
// DELETE:
func (c *Client) onCloseHooks() {
    c.lock.RLock()
    defer c.lock.RUnlock()
    for _, h := range c.closeHooks {
        h()
    }
}
```

### 5. Remove Tests (client_test.go:938-968)

- `TestClientOnClose`
- `TestClientOnCloseMultipleHooks`

## Migration Guide

Users relying on `OnClose()` should convert to inline cleanup:

```go
// BEFORE:
c.OnClose(func() {
    cleanupResource()
})
defer c.Close()

// AFTER:
defer cleanupResource()
```

## Implementation Order

1. Delete `CloseHook` type definition
2. Delete `closeHooks` field from `Client` struct
3. Delete `OnClose()` method
4. Delete `Close()` method
5. Delete `onCloseHooks()` private method
6. Delete test functions

## Verification

1. Compile check - ensure code compiles after deletion
2. Run test suite - ensure no other code depends on these methods
3. Grep for any remaining references to `OnClose`, `Close`, `closeHooks`, `CloseHook`
