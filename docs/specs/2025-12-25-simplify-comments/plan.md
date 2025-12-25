# Simplify Comments Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use evo-executing-plans to implement this plan task-by-task.

**Goal:** Remove copyright headers and simplify method comments across all Go files.

**Architecture:** Text-only changes to comments. No code behavior changes, no tests needed.

**Tech Stack:** Go, standard text editing

---

## Task 1: client.go - Remove Copyright Header

**Files:**
- Modify: `client.go:1-4`

**Step 1: Remove copyright header**

Delete lines 1-4:
```go
// Copyright (c) 2015-present Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
// SPDX-License-Identifier: MIT
```

Result should start with `package fetch`

**Step 2: Verify build still works**

Run: `go build ./...`
Expected: SUCCESS (no code changes, only comments)

---

## Task 2: client.go - Simplify Method Comments (Part 1: BaseURL, Header, Cookie, QueryParams, FormData)

**Files:**
- Modify: `client.go:133-271`

**Step 1: Simplify BaseURL methods**

Change line 133 from:
```go
// BaseURL method returns the Base URL value from the client instance.
```
To:
```go
// BaseURL returns the Base URL value.
```

Change line 140 from:
```go
// SetBaseURL sets the Base URL in the client instance.
```
To:
```go
// SetBaseURL sets the Base URL.
```

**Step 2: Simplify Header methods**

Change line 148 from:
```go
// Header method returns the headers from the client instance.
```
To:
```go
// Header returns the headers.
```

Change line 155 from:
```go
// SetHeader sets a single header and its value in the client instance.
```
To:
```go
// SetHeader sets a single header and its value.
```

Change line 163 from:
```go
// SetHeaders sets multiple headers and their values at once.
```
To:
```go
// SetHeaders sets multiple headers and their values.
```

Change line 173 from:
```go
// SetHeaderVerbatim sets the HTTP header key and value verbatim.
```
To:
```go
// SetHeaderVerbatim sets the header key and value verbatim.
```

**Step 3: Simplify Cookie methods**

Change line 196 from:
```go
// CookieJar method returns the HTTP cookie jar instance from the underlying Go HTTP Client.
```
To:
```go
// CookieJar returns the HTTP cookie jar instance.
```

Change line 209 from:
```go
// Cookies method returns all cookies registered in the client instance.
```
To:
```go
// Cookies returns all cookies.
```

Change line 216 from:
```go
// SetCookie appends a single cookie to the client instance.
```
To:
```go
// SetCookie appends a single cookie.
```

Change line 224 from:
```go
// SetCookies sets an array of cookies in the client instance.
```
To:
```go
// SetCookies sets multiple cookies.
```

**Step 4: Simplify QueryParams methods**

Change line 232 from:
```go
// QueryParams method returns all query parameters and their values from the client instance.
```
To:
```go
// QueryParams returns all query parameters.
```

Change line 239 from:
```go
// SetQueryParam sets a single parameter and its value in the client instance.
```
To:
```go
// SetQueryParam sets a single parameter and its value.
```

Change line 247 from:
```go
// SetQueryParams sets multiple parameters and their values at once.
```
To:
```go
// SetQueryParams sets multiple parameters.
```

**Step 5: Simplify FormData methods**

Change line 256 from:
```go
// FormData method returns the form parameters and their values from the client instance.
```
To:
```go
// FormData returns the form parameters.
```

Change line 263 from:
```go
// SetFormData sets form parameters and their values in the client instance.
```
To:
```go
// SetFormData sets form parameters.
```

**Step 6: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 3: client.go - Simplify Method Comments (Part 2: Middlewares, Hooks, ContentType, Debug)

**Files:**
- Modify: `client.go:273-551`

**Step 1: Simplify R() and NewRequest()**

Change line 273 from:
```go
// R creates a new request instance.
```
To:
```go
// R creates a new request.
```

Change line 308 from:
```go
// NewRequest is an alias for R.
```
(Keep as-is - already simple)

**Step 2: Simplify Middleware methods**

Change line 313 from:
```go
// SetRequestMiddlewares sets the request middlewares sequence.
```
To:
```go
// SetRequestMiddlewares sets the request middlewares.
```

Change line 321 from:
```go
// SetResponseMiddlewares sets the response middlewares sequence.
```
To:
```go
// SetResponseMiddlewares sets the response middlewares.
```

Change line 335 from:
```go
// AddRequestMiddleware appends a request middleware to the chain.
```
To:
```go
// AddRequestMiddleware appends a request middleware.
```

Change line 350 from:
```go
// AddResponseMiddleware appends response middleware to the chain.
```
To:
```go
// AddResponseMiddleware appends response middleware.
```

**Step 3: Simplify Hook methods**

Change line 358 from:
```go
// OnError adds a callback that will be run whenever a request execution fails.
```
To:
```go
// OnError adds a callback for request failures.
```

Change line 366 from:
```go
// OnSuccess adds a callback that will be run whenever a request execution succeeds.
```
To:
```go
// OnSuccess adds a callback for request success.
```

Change line 374 from:
```go
// OnInvalid adds a callback that will be run whenever a request execution fails before it starts.
```
To:
```go
// OnInvalid adds a callback for pre-execution failures.
```

Change line 382 from:
```go
// OnPanic adds a callback that will be run whenever a request execution panics.
```
To:
```go
// OnPanic adds a callback for request panics.
```

Change line 390 from:
```go
// OnClose adds a callback that will be run whenever the client is closed.
```
To:
```go
// OnClose adds a callback for client close.
```

**Step 4: Simplify ContentType methods**

Change line 398 from:
```go
// ContentTypeEncoders method returns all the registered content type encoders.
```
To:
```go
// ContentTypeEncoders returns all registered encoders.
```

Change line 405 from:
```go
// AddContentTypeEncoder adds a Content-Type encoder into the client.
```
To:
```go
// AddContentTypeEncoder adds a Content-Type encoder.
```

Change line 424 from:
```go
// ContentTypeDecoders method returns all the registered content type decoders.
```
To:
```go
// ContentTypeDecoders returns all registered decoders.
```

Change line 431 from:
```go
// AddContentTypeDecoder adds a Content-Type decoder into the client.
```
To:
```go
// AddContentTypeDecoder adds a Content-Type decoder.
```

**Step 5: Simplify Decompressor methods**

Change line 450 from:
```go
// ContentDecompressors method returns all the registered content-encoding Decompressors.
```
To:
```go
// ContentDecompressors returns all registered decompressors.
```

Change line 457 from:
```go
// AddContentDecompressor adds a Content-Encoding decompressor into the client.
```
To:
```go
// AddContentDecompressor adds a Content-Encoding decompressor.
```

Change line 468 from:
```go
// ContentDecompressorKeys returns all registered decompressor keys as a comma-separated string.
```
To:
```go
// ContentDecompressorKeys returns decompressor keys.
```

Change line 475 from:
```go
// SetContentDecompressorKeys sets the Content-Encoding directives order.
```
To:
```go
// SetContentDecompressorKeys sets decompressor priority order.
```

**Step 6: Simplify Debug methods**

Change line 491 from:
```go
// IsDebug returns true if the client is in debug mode.
```
To:
```go
// IsDebug returns the debug mode status.
```

Change line 498 from:
```go
// SetDebug enables debug mode on the client.
```
To:
```go
// SetDebug enables or disables debug mode.
```

Change line 506 from:
```go
// DebugBodyLimit returns the debug body limit value.
```
To:
```go
// DebugBodyLimit returns the debug body limit.
```

Change line 513 from:
```go
// SetDebugBodyLimit sets the maximum size for logging request/response body in debug mode.
```
To:
```go
// SetDebugBodyLimit sets the maximum body size for debug logging.
```

Change line 527 from:
```go
// OnDebugLog sets the debug log callback function.
```
To:
```go
// OnDebugLog sets the debug log callback.
```

Change line 539 from:
```go
// DebugLogFormatterFunc returns the debug log formatter.
```
To:
```go
// DebugLogFormatterFunc returns the debug log formatter. (Wait, this is a private method - no change needed)
```

Actually, line 539 `debugLogFormatterFunc()` is a private method. Skip it.

Change line 545 from:
```go
// SetDebugLogFormatter sets the debug log formatter.
```
To:
```go
// SetDebugLogFormatter sets the debug log formatter. (Already simple - keep)
```

**Step 7: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 4: client.go - Simplify Method Comments (Part 3: Logger, Settings, Redirect, Proxy, Output)

**Files:**
- Modify: `client.go:553-729`

**Step 1: Simplify Logger methods**

Change line 553 from:
```go
// IsDisableWarn returns true if warning messages are disabled.
```
To:
```go
// IsDisableWarn returns the warning disable status.
```

Change line 560 from:
```go
// SetDisableWarn disables warning log messages.
```
To:
```go
// SetDisableWarn enables or disables warning messages.
```

Change line 568 from:
```go
// Logger method returns the logger instance.
```
To:
```go
// Logger returns the logger instance. (Already simple)
```

Change line 575 from:
```go
// SetLogger sets the logger instance.
```
To:
```go
// SetLogger sets the logger. (Already simple)
```

**Step 2: Simplify ContentLength and Timeout methods**

Change line 583 from:
```go
// IsContentLength returns true if content length is set.
```
To:
```go
// IsContentLength returns the content length status.
```

Change line 590 from:
```go
// SetContentLength enables the HTTP header `Content-Length` for requests.
```
To:
```go
// SetContentLength enables the Content-Length header.
```

Change line 598 from:
```go
// Timeout returns the timeout duration.
```
To:
```go
// Timeout returns the timeout duration. (Already simple)
```

Change line 605 from:
```go
// SetTimeout sets the timeout for requests.
```
To:
```go
// SetTimeout sets the request timeout. (Already simple)
```

**Step 3: Simplify Error and Redirect methods**

Change line 613 from:
```go
// Error returns the common error object type.
```
To:
```go
// Error returns the error object type.
```

Change line 620 from:
```go
// SetError registers the common error object for automatic unmarshaling.
```
To:
```go
// SetError registers the error object for automatic unmarshaling.
```

Change line 636-644 SetRedirectPolicy - Keep as-is with usage examples (they help explain redirect policies)

**Step 4: Simplify Proxy methods**

Change line 659 from:
```go
// ProxyURL method returns the proxy URL if set otherwise nil.
```
To:
```go
// ProxyURL returns the proxy URL.
```

Change line 666-674 SetProxy - Remove usage examples, keep:
```go
// SetProxy sets the proxy URL.
```

Change line 695 from:
```go
// RemoveProxy method removes the proxy configuration from the Resty client
```
To:
```go
// RemoveProxy removes the proxy configuration.
```

**Step 5: Simplify OutputDirectory methods**

Change line 712 from:
```go
// OutputDirectory method returns the output directory value from the client.
```
To:
```go
// OutputDirectory returns the output directory.
```

Change line 719-728 SetOutputDirectory - Remove usage example, keep:
```go
// SetOutputDirectory sets the output directory for saving HTTP responses.
```

**Step 6: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 5: client.go - Simplify Method Comments (Part 4: Transport, Scheme, PathParams, JSON, BodyLimit, Trace)

**Files:**
- Modify: `client.go:731-1022`

**Step 1: Simplify Transport methods**

Change line 731-732 from:
```go
// HTTPTransport method does type assertion and returns [http.Transport]
// from the client instance, if type assertion fails it returns an error
```
To:
```go
// HTTPTransport returns the http.Transport.
```

Change line 742-743 from:
```go
// Transport method returns underlying client transport referance as-is
// i.e., [http.RoundTripper]
```
To:
```go
// Transport returns the underlying http.RoundTripper.
```

Change line 750-761 SetTransport - Remove usage example, keep:
```go
// SetTransport sets a custom http.RoundTripper.
// NOTE: It overwrites the existing transport.
```

**Step 2: Simplify Scheme methods**

Change line 771-773 Scheme - Remove usage example, keep:
```go
// Scheme returns the custom scheme value.
```

Change line 780-782 SetScheme - Remove usage example, keep:
```go
// SetScheme sets a custom scheme.
```

**Step 3: Simplify Connection and Parse methods**

Change line 792-795 SetCloseConnection - Keep as-is:
```go
// SetCloseConnection sets the Close field in HTTP request.
```

Change line 803-810 SetDoNotParseResponse - Keep NOTE, remove examples:
```go
// SetDoNotParseResponse disables automatic response parsing.
// NOTE: Default response middlewares are not executed.
```

**Step 4: Simplify PathParams methods**

Change line 818-820 PathParams - Remove example, keep:
```go
// PathParams returns the path parameters.
```

Change line 827-840 SetPathParam - Remove examples, keep:
```go
// SetPathParam sets a single URL path parameter.
// The value will be escaped using url.PathEscape.
```

Change line 848-865 SetPathParams - Remove examples, keep:
```go
// SetPathParams sets multiple URL path parameters.
// Values will be escaped using url.PathEscape.
```

Change line 873-886 SetRawPathParam - Remove example, keep:
```go
// SetRawPathParam sets a URL path parameter without escaping.
```

Change line 894-911 SetRawPathParams - Remove examples, keep:
```go
// SetRawPathParams sets multiple URL path parameters without escaping.
```

**Step 5: Simplify JSON and BodyLimit methods**

Change line 919-924 SetJSONEscapeHTML - Keep NOTE:
```go
// SetJSONEscapeHTML enables or disables HTML escape on JSON marshal.
// NOTE: Only applies to standard JSON Marshaller.
```

Change line 932-933 from:
```go
// ResponseBodyLimit method returns the value max body size limit in bytes from
// the client instance.
```
To:
```go
// ResponseBodyLimit returns the response body size limit.
```

Change line 940-950 SetResponseBodyLimit - Keep details and NOTE:
```go
// SetResponseBodyLimit sets the response body size limit.
// NOTE: Limit is not enforced when <= 0, or with SetOutputFileName, or DoNotParseResponse.
```

**Step 6: Simplify Trace and other methods**

Change line 958-967 SetTrace (EnableTrace comment) - Remove usage example, keep:
```go
// SetTrace enables HTTP trace for requests.
```

Change line 969-970 from:
```go
// IsTrace method returns true if the trace is enabled on the client instance; otherwise, it returns false.
```
To:
```go
// IsTrace returns the trace status.
```

Change line 976-979 SetTrace - Keep references:
```go
// SetTrace enables or disables tracing.
```

Change line 987-992 SetUnescapeQueryParams - Keep NOTE:
```go
// SetUnescapeQueryParams sets whether to unescape query parameters.
// NOTE: Request failure is possible with non-standard usage.
```

Change line 1000-1001 from:
```go
// ResponseBodyUnlimitedReads method returns true if enabled. Otherwise, it returns false
```
To:
```go
// ResponseBodyUnlimitedReads returns the unlimited reads status.
```

Change line 1007-1016 SetResponseBodyUnlimitedReads - Keep NOTEs:
```go
// SetResponseBodyUnlimitedReads enables unlimited response body reads.
// NOTE: Keeps response body in memory. Unlimited reads also work in debug mode.
```

**Step 7: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 6: client.go - Simplify Method Comments (Part 5: Clone, Close, etc.)

**Files:**
- Modify: `client.go:1024-1225`

**Step 1: Simplify remaining methods**

Change line 1024-1025 from:
```go
// IsProxySet method returns the true is proxy is set from the Resty client; otherwise
// false. By default, the proxy is set from the environment variable; refer to [http.ProxyFromEnvironment].
```
To:
```go
// IsProxySet returns whether proxy is configured.
```

Change line 1030 from:
```go
// Client method returns the underlying Go [http.Client] used by the Resty.
```
To:
```go
// Client returns the underlying http.Client.
```

Change line 1037-1044 Clone - Keep NOTEs:
```go
// Clone creates a shallow copy of the client.
// NOTE: Interface values are not deeply cloned. Not safe for concurrent use.
```

Change line 1077 from:
```go
// Close method performs cleanup and closure activities on the client instance
```
To:
```go
// Close performs cleanup activities.
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 7: client.go - Simplify Type Comments

**Files:**
- Modify: `client.go:66-84`

**Step 1: Simplify type alias comments**

Change line 67 from:
```go
// RequestMiddleware type is for request middleware, called before a request is sent
```
To: (remove comment entirely - self-documenting)

Change line 70 from:
```go
// ResponseMiddleware type is for response middleware, called after a response has been received
```
To: (remove comment entirely - self-documenting)

Change line 73 from:
```go
// ErrorHook type is for reacting to request errors, called after all retries were attempted
```
To:
```go
// ErrorHook is a callback for request errors.
```

Change line 76 from:
```go
// SuccessHook type is for reacting to request success
```
To:
```go
// SuccessHook is a callback for request success.
```

Change line 79 from:
```go
// CloseHook type is for reacting to client closing
```
To:
```go
// CloseHook is a callback for client close.
```

Change line 82 from:
```go
// RequestFunc type is for extended manipulation of the Request instance
```
To:
```go
// RequestFunc manipulates the Request.
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 8: client.go - Simplify Client Struct Comment

**Files:**
- Modify: `client.go:86`

**Step 1: Simplify struct comment**

Change line 86 from:
```go
// Client creates a Resty client with client-level settings that apply to all requests.
```
To:
```go
// Client is an HTTP client with configuration.
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 9: request.go - Remove Copyright Header

**Files:**
- Modify: `request.go:1-5`

**Step 1: Remove copyright header**

Delete lines 1-5:
```go
// Copyright (c) 2015-present Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
// SPDX-License-Identifier: MIT

package fetch
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 10: request.go - Simplify Struct and Method Comments

**Files:**
- Modify: `request.go`

**Step 1: Simplify Request struct comment**

Change line 29-31 from:
```go
// Request struct is used to compose and fire individual requests from
// Resty client. The [Request] provides an option to override client-level
// settings and also an option for the request composition.
```
To:
```go
// Request represents an HTTP request.
```

**Step 2: Simplify method comments (apply same pattern)**

For all methods, simplify:
- "method is used to" -> remove
- "in the client/request instance" -> remove
- "for the request" -> remove
- Usage examples -> remove
- Keep functional descriptions
- Keep NOTE/WARNING sections

Examples:
- `SetMethod method used to set` -> `SetMethod sets`
- `Context method returns` -> `Context returns`
- `SetContext method sets` -> `SetContext sets`

**Step 3: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 11: response.go - Remove Copyright and Simplify Comments

**Files:**
- Modify: `response.go`

**Step 1: Remove copyright header**

Delete lines 1-5 (copyright header)

**Step 2: Simplify Response struct comment**

Change line 22 from:
```go
// Response struct holds response values of executed requests.
```
To:
```go
// Response represents an HTTP response.
```

**Step 3: Simplify method comments**

Change line 38-40 from:
```go
// Status method returns the HTTP status string for the executed request.
//
//	Example: 200 OK
```
To:
```go
// Status returns the HTTP status string.
```

Change line 48-50 from:
```go
// StatusCode method returns the HTTP status code for the executed request.
//
//	Example: 200
```
To:
```go
// StatusCode returns the HTTP status code.
```

Apply same pattern to remaining methods.

**Step 4: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 12: resty.go - Remove Copyright and Simplify Comments

**Files:**
- Modify: `resty.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Simplify package comment**

Change line 6 from:
```go
// package fetch provides Simple HTTP, REST, and SSE client library for Go.
```
To:
```go
// Package fetch provides an HTTP client library.
```

**Step 3: Simplify function comments**

Change line 17 from:
```go
// New method creates a new Resty client.
```
To:
```go
// New creates a new Client.
```

Change line 22 from:
```go
// NewWithTransport method creates a new Resty client with the given [http.RoundTripper].
// If transport is nil, [http.DefaultTransport] is used.
```
To:
```go
// NewWithTransport creates a new Client with the given transport.
// Uses http.DefaultTransport if transport is nil.
```

Change line 34 from:
```go
// NewWithClient method creates a new Resty client with given [http.Client].
```
To:
```go
// NewWithClient creates a new Client with the given http.Client.
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 13: middleware.go - Remove Copyright and Simplify Comments

**Files:**
- Modify: `middleware.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Simplify PrepareRequestMiddleware comment**

Change line 25-27 from:
```go
// PrepareRequestMiddleware method is used to prepare HTTP requests from
// user provides request values. Request preparation fails if any error occurs
```
To:
```go
// PrepareRequestMiddleware prepares HTTP requests from user values.
```

**Step 3: Apply same pattern to remaining methods**

**Step 4: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 14: debug.go - Remove Copyright and Simplify Comments

**Files:**
- Modify: `debug.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Simplify type comments**

Change line 15-17 from:
```go
// DebugLogCallbackFunc function type is for request and response debug log callback purposes.
// It gets called before Resty logs it
```
To:
```go
// DebugLogCallbackFunc is called before debug logging.
```

Change line 19-21 from:
```go
// DebugLogFormatterFunc function type is used to implement debug log formatting.
// See out of the box [DebugLogStringFormatter], [DebugLogJSONFormatter]
```
To:
```go
// DebugLogFormatterFunc formats debug logs.
```

Change line 23-24 from:
```go
// DebugLog struct is used to collect details from Resty request and response
// for debug logging callback purposes.
```
To:
```go
// DebugLog holds request and response debug information.
```

Change line 31 from:
```go
// DebugLogRequest type used to capture debug info about the [Request].
```
To:
```go
// DebugLogRequest holds request debug information.
```

Change line 41 from:
```go
// DebugLogResponse type used to capture debug info about the [Response].
```
To:
```go
// DebugLogResponse holds response debug information.
```

**Step 3: Simplify function comments**

Change line 54-57 from:
```go
// DebugLogFormatter function formats the given debug log info in human readable
// format.
//
// This is the default debug log formatter in the Resty.
```
To:
```go
// DebugLogFormatter formats debug logs in human-readable format.
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 15: redirect.go - Remove Copyright and Simplify Comments

**Files:**
- Modify: `redirect.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Simplify type comments**

Change line 17-21 from:
```go
// RedirectPolicy to regulate the redirects in the Resty client.
// Objects implementing the [RedirectPolicy] interface can be registered as
//
// Apply function should return nil to continue the redirect journey; otherwise
// return error to stop the redirect.
```
To:
```go
// RedirectPolicy regulates client redirects.
// Apply returns nil to continue, error to stop.
```

Change line 26-28 from:
```go
// The [RedirectPolicyFunc] type is an adapter to allow the use of ordinary
// functions as [RedirectPolicy]. If `f` is a function with the appropriate
// signature, RedirectPolicyFunc(f) is a RedirectPolicy object that calls `f`.
```
To:
```go
// RedirectPolicyFunc adapts a function to RedirectPolicy.
```

Change line 31 from:
```go
// RedirectInfo struct is used to capture the URL and status code for the redirect history
```
To:
```go
// RedirectInfo captures redirect URL and status code.
```

**Step 3: Simplify function comments (remove usage examples)**

Change line 43-45 from:
```go
// NoRedirectPolicy is used to disable the redirects in the Resty client
//
//	resty.SetRedirectPolicy(resty.NoRedirectPolicy())
```
To:
```go
// NoRedirectPolicy disables redirects.
```

Change line 52-54 from:
```go
// FlexibleRedirectPolicy method is convenient for creating several redirect policies for Resty clients.
//
//	resty.SetRedirectPolicy(FlexibleRedirectPolicy(20))
```
To:
```go
// FlexibleRedirectPolicy creates a redirect policy with max redirects.
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 16: trace.go - Remove Copyright Header

**Files:**
- Modify: `trace.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 17: multipart.go - Remove Copyright Header

**Files:**
- Modify: `multipart.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 18: stream.go - Remove Copyright Header

**Files:**
- Modify: `stream.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 19: util.go - Remove Copyright Header

**Files:**
- Modify: `util.go`

**Step 1: Remove copyright header**

Delete lines 1-5

**Step 2: Verify build**

Run: `go build ./...`
Expected: SUCCESS

---

## Task 20: Test Files - Remove Copyright Headers

**Files:**
- Modify: All `*_test.go` files

**Step 1: Remove copyright headers from test files**

For each test file, delete the copyright header (lines 1-5):
- client_test.go
- request_test.go
- response_test.go (if exists)
- resty_test.go
- middleware_test.go
- multipart_test.go
- stream_test.go
- context_test.go
- util_test.go
- benchmark_test.go

**Step 2: Verify tests still pass**

Run: `go test ./...`
Expected: All tests PASS

---

## Task 21: Final Verification

**Files:**
- All project files

**Step 1: Verify no copyright headers remain**

Run: `grep -r "Copyright (c)" *.go`
Expected: No results (or only in LICENSE file if present)

**Step 2: Full build and test**

Run: `go build ./... && go test ./...`
Expected: All SUCCESS

**Step 3: Check diff**

Run: `git diff --stat`
Expected: ~200-300 lines removed (comments only)

---

## Notes

- No behavior changes, only comment modifications
- No new tests needed - existing tests verify behavior unchanged
- Git diff should only show comment removals
