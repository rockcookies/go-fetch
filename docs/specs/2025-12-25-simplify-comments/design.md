# Design: Simplify Comments

**Date:** 2025-12-25
**Status:** Proposed

## Problem Statement

The codebase contains verbose comments that add noise without meaningful value:

1. **Redundant copyright headers**: Every `.go` file has a 4-line MIT license preamble, duplicating the LICENSE file
2. **Usage examples in GoDoc**: Method comments include code examples that make documentation verbose and harder to scan

This violates the project's **Principle 1: Simplicity First** - specifically the "less is more" philosophy.

## Proposed Changes

### 1. Remove Copyright Headers

**Before:**
```go
// Copyright (c) 2015-present Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
// SPDX-License-Identifier: MIT

package fetch
```

**After:**
```go
package fetch
```

License remains only in the project root `LICENSE` file.

### 2. Simplify Method Comments

Remove usage examples, keep functional descriptions and critical notes.

**Before:**
```go
// SetProxy method sets the Proxy URL and Port for the Resty client.
//
//	// HTTP/HTTPS proxy
//	client.SetProxy("http://proxyserver:8888")
//
//	// SOCKS5 Proxy
//	client.SetProxy("socks5://127.0.0.1:1080")
//
// OR you could also set Proxy via environment variable, refer to [http.ProxyFromEnvironment]
func (c *Client) SetProxy(proxyURL string) *Client {
```

**After:**
```go
// SetProxy sets the proxy URL for the client.
func (c *Client) SetProxy(proxyURL string) *Client {
```

**Preserve critical notes:**
```go
// SetDoNotParseResponse instructs Resty not to parse the response body automatically.
// NOTE: The default response middlewares are not executed when using this option.
func (c *Client) SetDoNotParseResponse(notParse bool) *Client {
```

### 3. Type Aliases

Simplify type comments to single-line descriptions.

**Before:**
```go
// RequestMiddleware type is for request middleware, called before a request is sent
RequestMiddleware func(*Client, *Request) error
```

**After:**
```go
RequestMiddleware func(*Client, *Request) error
```

The function signature is self-documenting.

## Affected Files

All `.go` files in the project (approximately 20 files):

- client.go
- request.go
- response.go
- resty.go
- middleware.go
- debug.go
- redirect.go
- trace.go
- multipart.go
- stream.go
- util.go

## Expected Impact

- **Lines removed**: ~200-300 lines of comments
- **Readability**: Improved - code is easier to scan
- **GoDoc**: Cleaner auto-generated documentation
- **Maintenance**: Less noise to keep in sync

## Rationale

Following **Constitution Principle 3.3**: Comments should explain "why", not "what". Usage examples belong in:
- Test files (`*_test.go`)
- Separate documentation
- External tutorials

## Constitution Compliance

- **Principle 1.1 (YAGNI)**: Remove unnecessary documentation overhead
- **Principle 1.3 (Anti-Over-Engineering)**: Simple comments over verbose examples
- **Principle 3.3 (Meaningful Comments)**: Focus on "why" not "what"
