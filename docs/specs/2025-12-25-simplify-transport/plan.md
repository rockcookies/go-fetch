# Implementation Plan: Simplify Transport Initialization

## Overview

Remove TransportSettings and 4 constructors, add single NewWithTransport function.

## Tasks

### Phase 1: Write New Tests First (TDD)

- [ ] **Task 1:** Add `TestNewWithTransport` - verify nil transport uses http.DefaultTransport
  - File: `client_test.go`
  - Create client with `NewWithTransport(nil)`
  - Verify client's httpClient.Transport is http.DefaultTransport

- [ ] **Task 2:** Add `TestNewWithTransport_Custom` - verify custom transport is used
  - File: `client_test.go`
  - Create custom `*http.Transport` with specific settings
  - Create client with `NewWithTransport(customTransport)`
  - Verify client's httpClient.Transport is the custom one

- [ ] **Task 3:** Add `TestNewWithTransport_RealRequest` - verify custom transport works with real request
  - File: `client_test.go`
  - Create custom `*http.Transport` with modified settings
  - Create client with `NewWithTransport(customTransport)`
  - Make real HTTP request and verify it works

### Phase 2: Update Existing Tests

- [ ] **Task 4:** Update `TestCustomTransportSettings` to use new API
  - File: `client_test.go` (lines 509-533)
  - Replace `NewWithTransportSettings(customTransportSettings)` with manual `*http.Transport` creation + `NewWithTransport()`
  - Or delete entire test if covered by new tests

- [ ] **Task 5:** Update `TestDefaultDialerTransportSettings` subtests
  - File: `client_test.go` (lines 535-556)
  - Replace `NewWithTransportSettings(nil)` with `New()` or `NewWithTransport(nil)`
  - Replace `NewWithDialerAndTransportSettings(nil, nil)` with `New()` or `NewWithTransport(nil)`

- [ ] **Task 6:** Delete `TestNewWithDialer` entirely
  - File: `client_test.go` (lines 558-572)
  - Remove entire function

- [ ] **Task 7:** Delete `TestNewWithLocalAddr` entirely
  - File: `client_test.go` (lines 574-587)
  - Remove entire function

- [ ] **Task 8:** Update `TestTraceInfoOnTimeout`
  - File: `request_test.go` (line 1621)
  - Replace `NewWithTransportSettings(&TransportSettings{DialerTimeout: ...})`
  - Create `*http.Transport` with `DialContext` timeout manually
  - Use `NewWithTransport()`

- [ ] **Task 9:** Update `TestTraceInfoOnTimeoutWithSetTimeout` if affected
  - File: `request_test.go` (line 1643)
  - Check if it uses removed APIs

### Phase 3: Implement Core Changes

- [ ] **Task 10:** Add `NewWithTransport` function to resty.go
  - File: `resty.go`
  - Add after `NewWithClient` function (around line 39)
  - Implementation: check nil, use http.DefaultTransport, call NewWithClient

- [ ] **Task 11:** Modify `New` function in resty.go
  - File: `resty.go` (line 26-28)
  - Change from `return NewWithTransportSettings(nil)` to `return NewWithTransport(nil)`

- [ ] **Task 12:** Remove `NewWithTransportSettings` from resty.go
  - File: `resty.go` (lines 30-34)

- [ ] **Task 13:** Remove `NewWithDialer` from resty.go
  - File: `resty.go` (lines 41-45)

- [ ] **Task 14:** Remove `NewWithLocalAddr` from resty.go
  - File: `resty.go` (lines 47-53)

- [ ] **Task 15:** Remove `NewWithDialerAndTransportSettings` from resty.go
  - File: `resty.go` (lines 55-62)

- [ ] **Task 16:** Remove `createTransport` from resty.go
  - File: `resty.go` (lines 68-157)

- [ ] **Task 17:** Remove `TransportSettings` struct from client.go
  - File: `client.go` (lines 99-143)

- [ ] **Task 18:** Delete `transport_dial.go` file

- [ ] **Task 19:** Delete `transport_dial_wasm.go` file

### Phase 4: Run Tests and Verify

- [ ] **Task 20:** Run all tests to verify no regressions
  - `go test ./...`
  - Fix any issues found

- [ ] **Task 21:** Run tests with race detector
  - `go test -race ./...`

- [ ] **Task 22:** Build project to verify compilation
  - `go build ./...`

### Phase 5: Update Documentation

- [ ] **Task 23:** Check if any docs reference removed functions
  - Search for `NewWithTransportSettings`, `NewWithDialer`, `NewWithLocalAddr`, `NewWithDialerAndTransportSettings`, `TransportSettings`
  - Update or remove references

## Summary

- **Total Tasks:** 23
- **Estimated Lines Removed:** ~150-200
- **Estimated Lines Added:** ~50 (including tests)
- **Net Reduction:** ~100-150 lines

## Order of Execution

1. Tasks 1-3: Write new tests first (TDD)
2. Tasks 4-9: Update existing tests
3. Tasks 10-19: Implement core changes
4. Tasks 20-22: Run tests and verify
5. Task 23: Update documentation
