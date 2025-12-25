// Copyright (c) 2015-present Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
// SPDX-License-Identifier: MIT

package fetch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// MethodGet HTTP method
	MethodGet = "GET"

	// MethodPost HTTP method
	MethodPost = "POST"

	// MethodPut HTTP method
	MethodPut = "PUT"

	// MethodDelete HTTP method
	MethodDelete = "DELETE"

	// MethodPatch HTTP method
	MethodPatch = "PATCH"

	// MethodHead HTTP method
	MethodHead = "HEAD"

	// MethodOptions HTTP method
	MethodOptions = "OPTIONS"

	// MethodTrace HTTP method
	MethodTrace = "TRACE"
)

var (
	ErrNotHttpTransportType       = errors.New("resty: not a http.Transport type")
	ErrUnsupportedRequestBodyKind = errors.New("resty: unsupported request body kind")

	hdrUserAgentKey       = http.CanonicalHeaderKey("User-Agent")
	hdrAcceptKey          = http.CanonicalHeaderKey("Accept")
	hdrAcceptEncodingKey  = http.CanonicalHeaderKey("Accept-Encoding")
	hdrContentTypeKey     = http.CanonicalHeaderKey("Content-Type")
	hdrContentLengthKey   = http.CanonicalHeaderKey("Content-Length")
	hdrContentEncodingKey = http.CanonicalHeaderKey("Content-Encoding")
	hdrContentDisposition = http.CanonicalHeaderKey("Content-Disposition")
	hdrRetryAfterKey      = http.CanonicalHeaderKey("Retry-After")
	hdrCookieKey          = http.CanonicalHeaderKey("Cookie")

	plainTextType   = "text/plain; charset=utf-8"
	jsonContentType = "application/json"
	formContentType = "application/x-www-form-urlencoded"

	jsonKey = "json"
	xmlKey  = "xml"

	hdrUserAgentValue = "go-resty/" + Version + " (https://resty.dev)"
	bufPool           = &sync.Pool{New: func() any { return &bytes.Buffer{} }}
)

type (
	// RequestMiddleware type is for request middleware, called before a request is sent
	RequestMiddleware func(*Client, *Request) error

	// ResponseMiddleware type is for response middleware, called after a response has been received
	ResponseMiddleware func(*Client, *Response) error

	// ErrorHook type is for reacting to request errors, called after all retries were attempted
	ErrorHook func(*Request, error)

	// SuccessHook type is for reacting to request success
	SuccessHook func(*Client, *Response)

	// CloseHook type is for reacting to client closing
	CloseHook func()

	// RequestFunc type is for extended manipulation of the Request instance
	RequestFunc func(*Request) *Request
)

// Client struct is used to create a Resty client with client-level settings,
// these settings apply to all the requests raised from the client.
//
// Resty also provides an option to override most of the client settings
// at [Request] level.
type Client struct {
	lock                    *sync.RWMutex
	baseURL                 string
	queryParams             url.Values
	formData                url.Values
	pathParams              map[string]string
	header                  http.Header
	cookies                 []*http.Cookie
	errorType               reflect.Type
	debug                   bool
	disableWarn             bool
	timeout                 time.Duration
	responseBodyLimit       int64
	resBodyUnlimitedReads   bool
	jsonEscapeHTML          bool
	setContentLength        bool
	closeConnection         bool
	notParseResponse        bool
	isTrace                 bool
	debugBodyLimit          int
	outputDirectory         string
	scheme                  string
	log                     Logger
	ctx                     context.Context
	httpClient              *http.Client
	proxyURL                *url.URL
	debugLogFormatter       DebugLogFormatterFunc
	debugLogCallback        DebugLogCallbackFunc
	generateCurlCmd         bool
	debugLogCurlCmd         bool
	unescapeQueryParams     bool
	beforeRequest           []RequestMiddleware
	afterResponse           []ResponseMiddleware
	errorHooks              []ErrorHook
	invalidHooks            []ErrorHook
	panicHooks              []ErrorHook
	successHooks            []SuccessHook
	closeHooks              []CloseHook
	contentTypeEncoders     map[string]ContentTypeEncoder
	contentTypeDecoders     map[string]ContentTypeDecoder
	contentDecompressorKeys []string
	contentDecompressors    map[string]ContentDecompressor
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Client methods
//___________________________________

// BaseURL method returns the Base URL value from the client instance.
func (c *Client) BaseURL() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.baseURL
}

// SetBaseURL method sets the Base URL in the client instance. It will be used with a request
// raised from this client with a relative URL
//
//	// Setting HTTP address
//	client.SetBaseURL("http://myjeeva.com")
//
//	// Setting HTTPS address
//	client.SetBaseURL("https://myjeeva.com")
func (c *Client) SetBaseURL(url string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.baseURL = strings.TrimRight(url, "/")
	return c
}

// Header method returns the headers from the client instance.
func (c *Client) Header() http.Header {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.header
}

// SetHeader method sets a single header and its value in the client instance.
// These headers will be applied to all requests raised from the client instance.
// Also, it can be overridden by request-level header options.
//
// For Example: To set `Content-Type` and `Accept` as `application/json`
//
//	client.
//		SetHeader("Content-Type", "application/json").
//		SetHeader("Accept", "application/json")
//
// See [Request.SetHeader] or [Request.SetHeaders].
func (c *Client) SetHeader(header, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.header.Set(header, value)
	return c
}

// SetHeaders method sets multiple headers and their values at one go, and
// these headers will be applied to all requests raised from the client instance.
// Also, it can be overridden at request-level headers options.
//
// For Example: To set `Content-Type` and `Accept` as `application/json`
//
//	client.SetHeaders(map[string]string{
//		"Content-Type": "application/json",
//		"Accept": "application/json",
//	})
//
// See [Request.SetHeaders] or [Request.SetHeader].
func (c *Client) SetHeaders(headers map[string]string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	for h, v := range headers {
		c.header.Set(h, v)
	}
	return c
}

// SetHeaderVerbatim method is used to set the HTTP header key and value verbatim in the current request.
// It is typically helpful for legacy applications or servers that require HTTP headers in a certain way
//
// For Example: To set header key as `all_lowercase`, `UPPERCASE`, and `x-cloud-trace-id`
//
//	client.
//		SetHeaderVerbatim("all_lowercase", "available").
//		SetHeaderVerbatim("UPPERCASE", "available").
//		SetHeaderVerbatim("x-cloud-trace-id", "798e94019e5fc4d57fbb8901eb4c6cae")
//
// See [Request.SetHeaderVerbatim].
func (c *Client) SetHeaderVerbatim(header, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.header[header] = []string{value}
	return c
}

// Context method returns the [context.Context] from the client instance.
func (c *Client) Context() context.Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ctx
}

// SetContext method sets the given [context.Context] in the client instance and
// it gets added to [Request] raised from this instance.
func (c *Client) SetContext(ctx context.Context) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ctx = ctx
	return c
}

// CookieJar method returns the HTTP cookie jar instance from the underlying Go HTTP Client.
func (c *Client) CookieJar() http.CookieJar {
	return c.Client().Jar
}

// SetCookieJar method sets custom [http.CookieJar] in the resty client. It's a way to override the default.
//
// For Example, sometimes we don't want to save cookies in API mode so that we can remove the default
// CookieJar in resty client.
//
//	client.SetCookieJar(nil)
func (c *Client) SetCookieJar(jar http.CookieJar) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.httpClient.Jar = jar
	return c
}

// Cookies method returns all cookies registered in the client instance.
func (c *Client) Cookies() []*http.Cookie {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cookies
}

// SetCookie method appends a single cookie to the client instance.
// These cookies will be added to all the requests from this client instance.
//
//	client.SetCookie(&http.Cookie{
//		Name:"go-resty",
//		Value:"This is cookie value",
//	})
func (c *Client) SetCookie(hc *http.Cookie) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cookies = append(c.cookies, hc)
	return c
}

// SetCookies method sets an array of cookies in the client instance.
// These cookies will be added to all the requests from this client instance.
//
//	cookies := []*http.Cookie{
//		&http.Cookie{
//			Name:"go-resty-1",
//			Value:"This is cookie 1 value",
//		},
//		&http.Cookie{
//			Name:"go-resty-2",
//			Value:"This is cookie 2 value",
//		},
//	}
//
//	// Setting a cookies into resty
//	client.SetCookies(cookies)
func (c *Client) SetCookies(cs []*http.Cookie) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cookies = append(c.cookies, cs...)
	return c
}

// QueryParams method returns all query parameters and their values from the client instance.
func (c *Client) QueryParams() url.Values {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.queryParams
}

// SetQueryParam method sets a single parameter and its value in the client instance.
// It will be formed as a query string for the request.
//
//	For Example: `search=kitchen%20papers&size=large`
//
// In the URL after the `?` mark. These query params will be added to all the requests raised from
// this client instance. Also, it can be overridden at the request level.
//
// See [Request.SetQueryParam] or [Request.SetQueryParams].
//
//	client.
//		SetQueryParam("search", "kitchen papers").
//		SetQueryParam("size", "large")
func (c *Client) SetQueryParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.queryParams.Set(param, value)
	return c
}

// SetQueryParams method sets multiple parameters and their values at one go in the client instance.
// It will be formed as a query string for the request.
//
//	For Example: `search=kitchen%20papers&size=large`
//
// In the URL after the `?` mark. These query params will be added to all the requests raised from this
// client instance. Also, it can be overridden at the request level.
//
// See [Request.SetQueryParams] or [Request.SetQueryParam].
//
//	client.SetQueryParams(map[string]string{
//		"search": "kitchen papers",
//		"size": "large",
//	})
func (c *Client) SetQueryParams(params map[string]string) *Client {
	// Do not lock here since there is potential deadlock.
	for p, v := range params {
		c.SetQueryParam(p, v)
	}
	return c
}

// FormData method returns the form parameters and their values from the client instance.
func (c *Client) FormData() url.Values {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.formData
}

// SetFormData method sets Form parameters and their values in the client instance.
// The request content type would be set as `application/x-www-form-urlencoded`.
// The client-level form data gets added to all the requests. Also, it can be
// overridden at the request level.
//
// See [Request.SetFormData].
//
//	client.SetFormData(map[string]string{
//		"access_token": "BC594900-518B-4F7E-AC75-BD37F019E08F",
//		"user_id": "3455454545",
//	})
func (c *Client) SetFormData(data map[string]string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	for k, v := range data {
		c.formData.Set(k, v)
	}
	return c
}

// R method creates a new request instance; it's used for Get, Post, Put, Delete, Patch, Head, Options, etc.
func (c *Client) R() *Request {
	c.lock.RLock()
	defer c.lock.RUnlock()
	r := &Request{
		QueryParams:                url.Values{},
		FormData:                   url.Values{},
		Header:                     http.Header{},
		Cookies:                    make([]*http.Cookie, 0),
		PathParams:                 make(map[string]string),
		Timeout:                    c.timeout,
		Debug:                      c.debug,
		IsTrace:                    c.isTrace,
		CloseConnection:            c.closeConnection,
		DoNotParseResponse:         c.notParseResponse,
		DebugBodyLimit:             c.debugBodyLimit,
		ResponseBodyLimit:          c.responseBodyLimit,
		ResponseBodyUnlimitedReads: c.resBodyUnlimitedReads,

		client:              c,
		baseURL:             c.baseURL,
		multipartFields:     make([]*MultipartField, 0),
		jsonEscapeHTML:      c.jsonEscapeHTML,
		log:                 c.log,
		setContentLength:    c.setContentLength,
		generateCurlCmd:     c.generateCurlCmd,
		debugLogCurlCmd:     c.debugLogCurlCmd,
		unescapeQueryParams: c.unescapeQueryParams,
	}

	if c.ctx != nil {
		r.ctx = context.WithoutCancel(c.ctx) // refer to godoc for more info about this function
	}

	return r
}

// NewRequest method is an alias for method `R()`.
func (c *Client) NewRequest() *Request {
	return c.R()
}

// SetRequestMiddlewares method allows Resty users to override the default request
// middlewares sequence
//
//	client.SetRequestMiddlewares(
//		Custom1RequestMiddleware,
//		Custom2RequestMiddleware,
//		resty.PrepareRequestMiddleware, // after this, `Request.RawRequest` instance is available
//		Custom3RequestMiddleware,
//		Custom4RequestMiddleware,
//	)
//
// See, [Client.AddRequestMiddleware]
//
// NOTE:
//   - It overwrites the existing request middleware list.
//   - Be sure to include Resty request middlewares in the request chain at the appropriate spot.
func (c *Client) SetRequestMiddlewares(middlewares ...RequestMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.beforeRequest = middlewares
	return c
}

// SetResponseMiddlewares method allows Resty users to override the default response
// middlewares sequence
//
//	client.SetResponseMiddlewares(
//		Custom1ResponseMiddleware,
//		Custom2ResponseMiddleware,
//		resty.AutoParseResponseMiddleware, // before this, the body is not read except on the debug flow
//		Custom3ResponseMiddleware,
//		resty.SaveToFileResponseMiddleware, // See, Request.SetOutputFileName, Request.SetSaveResponse
//		Custom4ResponseMiddleware,
//		Custom5ResponseMiddleware,
//	)
//
// See, [Client.AddResponseMiddleware]
//
// NOTE:
//   - It overwrites the existing response middleware list.
//   - Be sure to include Resty response middlewares in the response chain at the appropriate spot.
func (c *Client) SetResponseMiddlewares(middlewares ...ResponseMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.afterResponse = middlewares
	return c
}

func (c *Client) requestMiddlewares() []RequestMiddleware {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.beforeRequest
}

// AddRequestMiddleware method appends a request middleware to the before request chain.
// After all requests, middlewares are applied, and the request is sent to the host server.
//
//	client.AddRequestMiddleware(func(c *resty.Client, r *resty.Request) error {
//		// Now you have access to the Client and Request instance
//		// manipulate it as per your need
//
//		return nil 	// if its successful otherwise return error
//	})
func (c *Client) AddRequestMiddleware(m RequestMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	idx := len(c.beforeRequest) - 1
	c.beforeRequest = slices.Insert(c.beforeRequest, idx, m)
	return c
}

func (c *Client) responseMiddlewares() []ResponseMiddleware {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.afterResponse
}

// AddResponseMiddleware method appends response middleware to the after-response chain.
// All the response middlewares are applied; once we receive a response
// from the host server.
//
//	client.AddResponseMiddleware(func(c *resty.Client, r *resty.Response) error {
//		// Now you have access to the Client and Response instance
//		// Also, you could access request via Response.Request i.e., r.Request
//		// manipulate it as per your need
//
//		return nil 	// if its successful otherwise return error
//	})
func (c *Client) AddResponseMiddleware(m ResponseMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.afterResponse = append(c.afterResponse, m)
	return c
}

// OnError method adds a callback that will be run whenever a request execution fails.
// This is called after all retries have been attempted (if any).
// If there was a response from the server, the error will be wrapped in [ResponseError]
// which has the last response received from the server.
//
//	client.OnError(func(req *resty.Request, err error) {
//		if v, ok := err.(*resty.ResponseError); ok {
//			// Do something with v.Response
//		}
//		// Log the error, increment a metric, etc...
//	})
//
// Out of the [Client.OnSuccess], [Client.OnError], [Client.OnInvalid], [Client.OnPanic]
// callbacks, exactly one set will be invoked for each call to [Request.Execute] that completes.
//
// NOTE:
//   - Do not use [Client] setter methods within OnError hooks; deadlock will happen.
func (c *Client) OnError(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.errorHooks = append(c.errorHooks, h)
	return c
}

// OnSuccess method adds a callback that will be run whenever a request execution
// succeeds.  This is called after all retries have been attempted (if any).
//
// Out of the [Client.OnSuccess], [Client.OnError], [Client.OnInvalid], [Client.OnPanic]
// callbacks, exactly one set will be invoked for each call to [Request.Execute] that completes.
//
// NOTE:
//   - Do not use [Client] setter methods within OnSuccess hooks; deadlock will happen.
func (c *Client) OnSuccess(h SuccessHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.successHooks = append(c.successHooks, h)
	return c
}

// OnInvalid method adds a callback that will be run whenever a request execution
// fails before it starts because the request is invalid.
//
// Out of the [Client.OnSuccess], [Client.OnError], [Client.OnInvalid], [Client.OnPanic]
// callbacks, exactly one set will be invoked for each call to [Request.Execute] that completes.
//
// NOTE:
//   - Do not use [Client] setter methods within OnInvalid hooks; deadlock will happen.
func (c *Client) OnInvalid(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.invalidHooks = append(c.invalidHooks, h)
	return c
}

// OnPanic method adds a callback that will be run whenever a request execution
// panics.
//
// Out of the [Client.OnSuccess], [Client.OnError], [Client.OnInvalid], [Client.OnPanic]
// callbacks, exactly one set will be invoked for each call to [Request.Execute] that completes.
//
// If an [Client.OnSuccess], [Client.OnError], or [Client.OnInvalid] callback panics,
// then exactly one rule can be violated.
//
// NOTE:
//   - Do not use [Client] setter methods within OnPanic hooks; deadlock will happen.
func (c *Client) OnPanic(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.panicHooks = append(c.panicHooks, h)
	return c
}

// OnClose method adds a callback that will be run whenever the client is closed.
// The hooks are executed in the order they were registered.
func (c *Client) OnClose(h CloseHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.closeHooks = append(c.closeHooks, h)
	return c
}

// ContentTypeEncoders method returns all the registered content type encoders.
func (c *Client) ContentTypeEncoders() map[string]ContentTypeEncoder {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentTypeEncoders
}

// AddContentTypeEncoder method adds the user-provided Content-Type encoder into a client.
//
// NOTE: It overwrites the encoder function if the given Content-Type key already exists.
func (c *Client) AddContentTypeEncoder(ct string, e ContentTypeEncoder) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.contentTypeEncoders[ct] = e
	return c
}

func (c *Client) inferContentTypeEncoder(ct ...string) (ContentTypeEncoder, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, v := range ct {
		if d, f := c.contentTypeEncoders[v]; f {
			return d, f
		}
	}
	return nil, false
}

// ContentTypeDecoders method returns all the registered content type decoders.
func (c *Client) ContentTypeDecoders() map[string]ContentTypeDecoder {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentTypeDecoders
}

// AddContentTypeDecoder method adds the user-provided Content-Type decoder into a client.
//
// NOTE: It overwrites the decoder function if the given Content-Type key already exists.
func (c *Client) AddContentTypeDecoder(ct string, d ContentTypeDecoder) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.contentTypeDecoders[ct] = d
	return c
}

func (c *Client) inferContentTypeDecoder(ct ...string) (ContentTypeDecoder, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, v := range ct {
		if d, f := c.contentTypeDecoders[v]; f {
			return d, f
		}
	}
	return nil, false
}

// ContentDecompressors method returns all the registered content-encoding Decompressors.
func (c *Client) ContentDecompressors() map[string]ContentDecompressor {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentDecompressors
}

// AddContentDecompressor method adds the user-provided Content-Encoding ([RFC 9110]) Decompressor
// and directive into a client.
//
// NOTE: It overwrites the Decompressor function if the given Content-Encoding directive already exists.
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110
func (c *Client) AddContentDecompressor(k string, d ContentDecompressor) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !slices.Contains(c.contentDecompressorKeys, k) {
		c.contentDecompressorKeys = slices.Insert(c.contentDecompressorKeys, 0, k)
	}
	c.contentDecompressors[k] = d
	return c
}

// ContentDecompressorKeys method returns all the registered content-encoding Decompressors
// keys as comma-separated string.
func (c *Client) ContentDecompressorKeys() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return strings.Join(c.contentDecompressorKeys, ", ")
}

// SetContentDecompressorKeys method sets given Content-Encoding ([RFC 9110]) directives into the client instance.
//
// It checks the given Content-Encoding exists in the [ContentDecompressor] list before assigning it,
// if it does not exist, it will skip that directive.
//
// Use this method to overwrite the default order. If a new content Decompressor is added,
// that directive will be the first.
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110
func (c *Client) SetContentDecompressorKeys(keys []string) *Client {
	result := make([]string, 0)
	decoders := c.ContentDecompressors()
	for _, k := range keys {
		if _, f := decoders[k]; f {
			result = append(result, k)
		}
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.contentDecompressorKeys = result
	return c
}

// IsDebug method returns `true` if the client is in debug mode; otherwise, it is `false`.
func (c *Client) IsDebug() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debug
}

// SetDebug method enables the debug mode on the Resty client. The client logs details
// of every request and response.
//
//	client.SetDebug(true)
//
// Also, it can be enabled at the request level for a particular request; see [Request.SetDebug].
//   - For [Request], it logs information such as HTTP verb, Relative URL path,
//     Host, Headers, and Body if it has one.
//   - For [Response], it logs information such as Status, Response Time, Headers,
//     and Body if it has one.
func (c *Client) SetDebug(d bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debug = d
	return c
}

// DebugBodyLimit method returns the debug body limit value set on the client instance
func (c *Client) DebugBodyLimit() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debugBodyLimit
}

// SetDebugBodyLimit sets the maximum size in bytes for which the response and
// request body will be logged in debug mode.
//
//	client.SetDebugBodyLimit(1000000)
func (c *Client) SetDebugBodyLimit(sl int) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debugBodyLimit = sl
	return c
}

func (c *Client) debugLogCallbackFunc() DebugLogCallbackFunc {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debugLogCallback
}

// OnDebugLog method sets the debug log callback function to the client instance.
// Registered callback gets called before the Resty logs the information.
func (c *Client) OnDebugLog(dlc DebugLogCallbackFunc) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.debugLogCallback != nil {
		c.log.Warnf("Overwriting an existing on-debug-log callback from=%s to=%s",
			functionName(c.debugLogCallback), functionName(dlc))
	}
	c.debugLogCallback = dlc
	return c
}

func (c *Client) debugLogFormatterFunc() DebugLogFormatterFunc {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debugLogFormatter
}

// SetDebugLogFormatter method sets the Resty debug log formatter to the client instance.
func (c *Client) SetDebugLogFormatter(df DebugLogFormatterFunc) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debugLogFormatter = df
	return c
}

// IsDisableWarn method returns `true` if the warning message is disabled; otherwise, it is `false`.
func (c *Client) IsDisableWarn() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.disableWarn
}

// SetDisableWarn method disables the warning log message on the Resty client.
//
// For example, Resty warns users when BasicAuth is used in non-TLS mode.
//
//	client.SetDisableWarn(true)
func (c *Client) SetDisableWarn(d bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.disableWarn = d
	return c
}

// Logger method returns the logger instance used by the client instance.
func (c *Client) Logger() Logger {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.log
}

// SetLogger method sets given writer for logging Resty request and response details.
//
// Compliant to interface [resty.Logger]
func (c *Client) SetLogger(l Logger) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.log = l
	return c
}

// IsContentLength method returns true if the user requests to set content length. Otherwise, it is false.
func (c *Client) IsContentLength() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.setContentLength
}

// SetContentLength method enables the HTTP header `Content-Length` value for every request.
// By default, Resty won't set `Content-Length`.
//
//	client.SetContentLength(true)
//
// Also, you have the option to enable a particular request. See [Request.SetContentLength]
func (c *Client) SetContentLength(l bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.setContentLength = l
	return c
}

// Timeout method returns the timeout duration value from the client
func (c *Client) Timeout() time.Duration {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.timeout
}

// SetTimeout method is used to set a timeout for a request raised by the client.
//
//	client.SetTimeout(1 * time.Minute)
//
// It can be overridden at the request level. See [Request.SetTimeout]
//
// NOTE: Resty uses [context.WithTimeout] on the request, it does not use [http.Client].Timeout
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.timeout = timeout
	return c
}

// Error method returns the global or client common `Error` object type registered in the Resty.
func (c *Client) Error() reflect.Type {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.errorType
}

// SetError method registers the global or client common `Error` object into Resty.
// It is used for automatic unmarshalling if the response status code is greater than 399 and
// content type is JSON or XML. It can be a pointer or a non-pointer.
//
//	client.SetError(&Error{})
//	// OR
//	client.SetError(Error{})
func (c *Client) SetError(v any) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.errorType = inferType(v)
	return c
}

func (c *Client) newErrorInterface() any {
	e := c.Error()
	if e == nil {
		return e
	}
	return reflect.New(e).Interface()
}

// SetRedirectPolicy method sets the redirect policy for the client. Resty provides ready-to-use
// redirect policies. Wanna create one for yourself, refer to `redirect.go`.
//
//	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(20))
//
//	// Need multiple redirect policies together
//	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(20), resty.DomainCheckRedirectPolicy("host1.com", "host2.net"))
//
// NOTE: It overwrites the previous redirect policies in the client instance.
func (c *Client) SetRedirectPolicy(policies ...RedirectPolicy) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		for _, p := range policies {
			if err := p.Apply(req, via); err != nil {
				return err
			}
		}
		return nil // looks good, go ahead
	}
	return c
}

// ProxyURL method returns the proxy URL if set otherwise nil.
func (c *Client) ProxyURL() *url.URL {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.proxyURL
}

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

// RemoveProxy method removes the proxy configuration from the Resty client
//
//	client.RemoveProxy()
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

// OutputDirectory method returns the output directory value from the client.
func (c *Client) OutputDirectory() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.outputDirectory
}

// SetOutputDirectory method sets the output directory for saving HTTP responses in a file.
// Resty creates one if the output directory does not exist. This setting is optional,
// if you plan to use the absolute path in [Request.SetOutputFileName] and can used together.
//
//	client.SetOutputDirectory("/save/http/response/here")
func (c *Client) SetOutputDirectory(dirPath string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.outputDirectory = dirPath
	return c
}

// HTTPTransport method does type assertion and returns [http.Transport]
// from the client instance, if type assertion fails it returns an error
func (c *Client) HTTPTransport() (*http.Transport, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		return transport, nil
	}
	return nil, ErrNotHttpTransportType
}

// Transport method returns underlying client transport referance as-is
// i.e., [http.RoundTripper]
func (c *Client) Transport() http.RoundTripper {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.httpClient.Transport
}

// SetTransport method sets custom [http.Transport] or any [http.RoundTripper]
// compatible interface implementation in the Resty client.
//
//	transport := &http.Transport{
//		// something like Proxying to httptest.Server, etc...
//		Proxy: func(req *http.Request) (*url.URL, error) {
//			return url.Parse(server.URL)
//		},
//	}
//	client.SetTransport(transport)
//
// NOTE: It overwrites the Resty client transport instance and its configurations.
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if transport != nil {
		c.httpClient.Transport = transport
	}
	return c
}

// Scheme method returns custom scheme value from the client.
//
//	scheme := client.Scheme()
func (c *Client) Scheme() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.scheme
}

// SetScheme method sets a custom scheme for the Resty client. It's a way to override the default.
//
//	client.SetScheme("http")
func (c *Client) SetScheme(scheme string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !isStringEmpty(scheme) {
		c.scheme = strings.TrimSpace(scheme)
	}
	return c
}

// SetCloseConnection method sets variable `Close` in HTTP request struct with the given
// value. More info: https://golang.org/src/net/http/request.go
//
// It can be overridden at the request level, see [Request.SetCloseConnection]
func (c *Client) SetCloseConnection(close bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.closeConnection = close
	return c
}

// SetDoNotParseResponse method instructs Resty not to parse the response body automatically.
//
// Resty exposes the raw response body as [io.ReadCloser]. If you use it, do not
// forget to close the body, otherwise, you might get into connection leaks, and connection
// reuse may not happen.
//
// NOTE: The default [Response] middlewares are not executed when using this option. User
// takes over the control of handling response body from Resty.
func (c *Client) SetDoNotParseResponse(notParse bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.notParseResponse = notParse
	return c
}

// PathParams method returns the path parameters from the client.
//
//	pathParams := client.PathParams()
func (c *Client) PathParams() map[string]string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.pathParams
}

// SetPathParam method sets a single URL path key-value pair in the
// Resty client instance.
//
//	client.SetPathParam("userId", "sample@sample.com")
//
//	Result:
//	   URL - /v1/users/{userId}/details
//	   Composed URL - /v1/users/sample@sample.com/details
//
// It replaces the value of the key while composing the request URL.
// The value will be escaped using [url.PathEscape] function.
//
// It can be overridden at the request level,
// see [Request.SetPathParam] or [Request.SetPathParams]
func (c *Client) SetPathParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pathParams[param] = url.PathEscape(value)
	return c
}

// SetPathParams method sets multiple URL path key-value pairs at one go in the
// Resty client instance.
//
//	client.SetPathParams(map[string]string{
//		"userId":       "sample@sample.com",
//		"subAccountId": "100002",
//		"path":         "groups/developers",
//	})
//
//	Result:
//	   URL - /v1/users/{userId}/{subAccountId}/{path}/details
//	   Composed URL - /v1/users/sample@sample.com/100002/groups%2Fdevelopers/details
//
// It replaces the value of the key while composing the request URL.
// The values will be escaped using [url.PathEscape] function.
//
// It can be overridden at the request level,
// see [Request.SetPathParam] or [Request.SetPathParams]
func (c *Client) SetPathParams(params map[string]string) *Client {
	for p, v := range params {
		c.SetPathParam(p, v)
	}
	return c
}

// SetRawPathParam method sets a single URL path key-value pair in the
// Resty client instance without path escape.
//
//	client.SetRawPathParam("path", "groups/developers")
//
//	Result:
//		URL - /v1/users/{path}/details
//		Composed URL - /v1/users/groups/developers/details
//
// It replaces the value of the key while composing the request URL.
// The value will be used as-is, no path escape applied.
//
// It can be overridden at the request level,
// see [Request.SetRawPathParam] or [Request.SetRawPathParams]
func (c *Client) SetRawPathParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pathParams[param] = value
	return c
}

// SetRawPathParams method sets multiple URL path key-value pairs at one go in the
// Resty client instance without path escape.
//
//	client.SetRawPathParams(map[string]string{
//		"userId":       "sample@sample.com",
//		"subAccountId": "100002",
//		"path":         "groups/developers",
//	})
//
//	Result:
//	   URL - /v1/users/{userId}/{subAccountId}/{path}/details
//	   Composed URL - /v1/users/sample@sample.com/100002/groups/developers/details
//
// It replaces the value of the key while composing the request URL.
// The value will be used as-is, no path escape applied.
//
// It can be overridden at the request level,
// see [Request.SetRawPathParam] or [Request.SetRawPathParams]
func (c *Client) SetRawPathParams(params map[string]string) *Client {
	for p, v := range params {
		c.SetRawPathParam(p, v)
	}
	return c
}

// SetJSONEscapeHTML method enables or disables the HTML escape on JSON marshal.
// By default, escape HTML is `true`.
//
// NOTE: This option only applies to the standard JSON Marshaller used by Resty.
//
// It can be overridden at the request level, see [Request.SetJSONEscapeHTML]
func (c *Client) SetJSONEscapeHTML(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.jsonEscapeHTML = b
	return c
}

// ResponseBodyLimit method returns the value max body size limit in bytes from
// the client instance.
func (c *Client) ResponseBodyLimit() int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.responseBodyLimit
}

// SetResponseBodyLimit method sets a maximum body size limit in bytes on response,
// avoid reading too much data to memory.
//
// Client will return [resty.ErrResponseBodyTooLarge] if the body size of the body
// in the uncompressed response is larger than the limit.
// Body size limit will not be enforced in the following cases:
//   - ResponseBodyLimit <= 0, which is the default behavior.
//   - [Request.SetOutputFileName] is called to save response data to the file.
//   - "DoNotParseResponse" is set for client or request.
//
// It can be overridden at the request level; see [Request.SetResponseBodyLimit]
func (c *Client) SetResponseBodyLimit(v int64) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.responseBodyLimit = v
	return c
}

// SetTrace method enables the Resty client trace for the requests fired from
// the client using [httptrace.ClientTrace] and provides insights.
//
//	client := resty.New().SetTrace(true)
//
//	resp, err := client.R().Get("https://httpbin.org/get")
//	fmt.Println("error:", err)
//	fmt.Println("Trace Info:", resp.Request.TraceInfo())
//
// The method [Request.EnableTrace] is also available to get trace info for a single request.

// IsTrace method returns true if the trace is enabled on the client instance; otherwise, it returns false.
func (c *Client) IsTrace() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.isTrace
}

// SetTrace method is used to turn on/off the trace capability in the Resty client
// Refer to [Client.EnableTrace] or [Client.DisableTrace].
//
// Also, see [Request.SetTrace]
func (c *Client) SetTrace(t bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.isTrace = t
	return c
}

// EnableGenerateCurlCmd method enables the generation of curl command at the
// client instance level.
//
// By default, Resty does not log the curl command in the debug log since it has the potential
// to leak sensitive data unless explicitly enabled via [Client.SetDebugLogCurlCmd] or
// [Request.SetDebugLogCurlCmd].
//
// NOTE: Use with care.
//   - Potential to leak sensitive data from [Request] and [Response] in the debug log
//     when the debug log option is enabled.
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
//     when the debug log option is enabled.
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

// SetUnescapeQueryParams method sets the choice of unescape query parameters for the request URL.
// To prevent broken URL, Resty replaces space (" ") with "+" in the query parameters.
//
// See [Request.SetUnescapeQueryParams]
//
// NOTE: Request failure is possible due to non-standard usage of Unescaped Query Parameters.
func (c *Client) SetUnescapeQueryParams(unescape bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.unescapeQueryParams = unescape
	return c
}

// ResponseBodyUnlimitedReads method returns true if enabled. Otherwise, it returns false
func (c *Client) ResponseBodyUnlimitedReads() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.resBodyUnlimitedReads
}

// SetResponseBodyUnlimitedReads method is to turn on/off the response body in memory
// that provides an ability to do unlimited reads.
//
// It can be overridden at the request level; see [Request.SetResponseBodyUnlimitedReads]
//
// Unlimited reads are possible in a few scenarios, even without enabling it.
//   - When debug mode is enabled
//
// NOTE: Use with care
//   - Turning on this feature keeps the response body in memory, which might cause additional memory usage.
func (c *Client) SetResponseBodyUnlimitedReads(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.resBodyUnlimitedReads = b
	return c
}

// IsProxySet method returns the true is proxy is set from the Resty client; otherwise
// false. By default, the proxy is set from the environment variable; refer to [http.ProxyFromEnvironment].
func (c *Client) IsProxySet() bool {
	return c.ProxyURL() != nil
}

// Client method returns the underlying Go [http.Client] used by the Resty.
func (c *Client) Client() *http.Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.httpClient
}

// Clone method returns a clone of the original client.
//
// NOTE: Use with care:
//   - Interface values are not deeply cloned. Thus, both the original and the
//     clone will use the same value.
//   - It is not safe for concurrent use. You should only use this method
//     when you are sure that any other concurrent process is not using the client
//     or client instance is protected by a mutex.
func (c *Client) Clone(ctx context.Context) *Client {
	cc := new(Client)
	// dereference the pointer and copy the value
	*cc = *c

	cc.ctx = ctx
	cc.queryParams = cloneURLValues(c.queryParams)
	cc.formData = cloneURLValues(c.formData)
	cc.header = c.header.Clone()
	cc.pathParams = maps.Clone(c.pathParams)

	cc.contentTypeEncoders = maps.Clone(c.contentTypeEncoders)
	cc.contentTypeDecoders = maps.Clone(c.contentTypeDecoders)
	cc.contentDecompressors = maps.Clone(c.contentDecompressors)
	copy(cc.contentDecompressorKeys, c.contentDecompressorKeys)

	if c.proxyURL != nil {
		cc.proxyURL, _ = url.Parse(c.proxyURL.String())
	}
	// clone cookies
	if l := len(c.cookies); l > 0 {
		cc.cookies = make([]*http.Cookie, 0, l)
		for _, cookie := range c.cookies {
			cc.cookies = append(cc.cookies, cloneCookie(cookie))
		}
	}

	// certain values need to be reset
	cc.lock = &sync.RWMutex{}
	return cc
}

// Close method performs cleanup and closure activities on the client instance
func (c *Client) Close() error {
	// Execute close hooks first
	c.onCloseHooks()

	return nil
}

func (c *Client) executeRequestMiddlewares(req *Request) (err error) {
	for _, f := range c.requestMiddlewares() {
		if err = f(c, req); err != nil {
			return err
		}
	}
	return nil
}

// Executes method executes the given `Request` object and returns
// response or error.
func (c *Client) execute(req *Request) (*Response, error) {
	if err := c.executeRequestMiddlewares(req); err != nil {
		return nil, err
	}

	if hostHeader := req.Header.Get("Host"); hostHeader != "" {
		req.RawRequest.Host = hostHeader
	}

	prepareRequestDebugInfo(c, req)

	req.Time = time.Now()
	resp, err := c.Client().Do(req.withTimeout())

	response := &Response{Request: req, RawResponse: resp}
	response.setReceivedAt()
	if err != nil {
		return response, err
	}
	if req.multipartErrChan != nil {
		if err = <-req.multipartErrChan; err != nil {
			return response, err
		}
	}
	if resp != nil {
		response.Body = resp.Body
		if err = response.wrapContentDecompressor(); err != nil {
			return response, err
		}

		response.wrapLimitReadCloser()
	}

	if !req.DoNotParseResponse {
		if req.ResponseBodyUnlimitedReads || req.Debug {
			response.wrapCopyReadCloser()

			if err = response.readAll(); err != nil {
				return response, err
			}
		}
	}

	debugLogger(c, response)

	// Apply Response middleware
	for _, f := range c.responseMiddlewares() {
		if err = f(c, response); err != nil {
			response.Err = wrapErrors(err, response.Err)
		}
	}

	err = response.Err
	return response, err
}

// just an internal helper method
func (c *Client) outputLogTo(w io.Writer) *Client {
	c.Logger().(*logger).l.SetOutput(w)
	return c
}

// ResponseError is a wrapper that includes the server response with an error.
// Neither the err nor the response should be nil.
type ResponseError struct {
	Response *Response
	Err      error
}

func (e *ResponseError) Error() string {
	return e.Err.Error()
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}

// Helper to run errorHooks hooks.
// It wraps the error in a [ResponseError] if the resp is not nil
// so hooks can access it.
func (c *Client) onErrorHooks(req *Request, res *Response, err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if err != nil {
		if res != nil { // wrap with ResponseError
			err = &ResponseError{Response: res, Err: err}
		}
		for _, h := range c.errorHooks {
			h(req, err)
		}
	} else {
		for _, h := range c.successHooks {
			h(c, res)
		}
	}
}

// Helper to run panicHooks hooks.
func (c *Client) onPanicHooks(req *Request, err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, h := range c.panicHooks {
		h(req, err)
	}
}

// Helper to run invalidHooks hooks.
func (c *Client) onInvalidHooks(req *Request, err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, h := range c.invalidHooks {
		h(req, err)
	}
}

// Helper to run closeHooks hooks.
func (c *Client) onCloseHooks() {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, h := range c.closeHooks {
		h()
	}
}

func (c *Client) debugf(format string, v ...any) {
	if c.IsDebug() {
		c.Logger().Debugf(format, v...)
	}
}
