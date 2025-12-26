package fetch

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"
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
	ErrUnsupportedRequestBodyKind = errors.New("resty: unsupported request body kind")

	hdrAcceptKey          = http.CanonicalHeaderKey("Accept")
	hdrAcceptEncodingKey  = http.CanonicalHeaderKey("Accept-Encoding")
	hdrContentTypeKey     = http.CanonicalHeaderKey("Content-Type")
	hdrContentLengthKey   = http.CanonicalHeaderKey("Content-Length")
	hdrContentEncodingKey = http.CanonicalHeaderKey("Content-Encoding")
	hdrContentDisposition = http.CanonicalHeaderKey("Content-Disposition")
	hdrCookieKey          = http.CanonicalHeaderKey("Cookie")

	plainTextType   = "text/plain; charset=utf-8"
	jsonContentType = "application/json"
	formContentType = "application/x-www-form-urlencoded"

	jsonKey = "json"
	xmlKey  = "xml"

	bufPool = &sync.Pool{New: func() any { return &bytes.Buffer{} }}
)

type (
	RequestMiddleware func(*Client, *Request) error

	ResponseMiddleware func(*Client, *Response) error

	// ErrorHook is a callback for request errors.
	ErrorHook func(*Request, error)

	// SuccessHook is a callback for request success.
	SuccessHook func(*Client, *Response)

	// RequestFunc manipulates the Request.
	RequestFunc func(*Request) *Request
)

// Client is an HTTP client with configuration.
type Client struct {
	baseURL                 string
	queryParams             url.Values
	formData                url.Values
	pathParams              map[string]string
	header                  http.Header
	cookies                 []*http.Cookie
	debug                   bool
	responseBodyLimit       int64
	setContentLength        bool
	closeConnection         bool
	notParseResponse        bool
	isTrace                 bool
	debugBodyLimit          int
	scheme                  string
	httpClient              *http.Client
	debugLogCallback        DebugLogCallbackFunc
	beforeRequest           []RequestMiddleware
	afterResponse           []ResponseMiddleware
	contentTypeEncoders     map[string]ContentTypeEncoder
	contentTypeDecoders     map[string]ContentTypeDecoder
	contentDecompressorKeys []string
	contentDecompressors    map[string]ContentDecompressor
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Client methods
//___________________________________

// BaseURL returns the Base URL value.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// SetBaseURL sets the Base URL.
func (c *Client) SetBaseURL(url string) *Client {
	c.baseURL = strings.TrimRight(url, "/")
	return c
}

// Header returns the headers.
func (c *Client) Header() http.Header {
	return c.header
}

// SetHeader sets a single header and its value.
func (c *Client) SetHeader(header, value string) *Client {
	c.header.Set(header, value)
	return c
}

// SetHeaders sets multiple headers and their values.
func (c *Client) SetHeaders(headers map[string]string) *Client {
	for h, v := range headers {
		c.header.Set(h, v)
	}
	return c
}

// SetHeaderVerbatim sets the header key and value verbatim.
func (c *Client) SetHeaderVerbatim(header, value string) *Client {
	c.header[header] = []string{value}
	return c
}

// CookieJar returns the HTTP cookie jar instance.
func (c *Client) CookieJar() http.CookieJar {
	return c.httpClient.Jar
}

// SetCookieJar sets a custom cookie jar.
func (c *Client) SetCookieJar(jar http.CookieJar) *Client {
	c.httpClient.Jar = jar
	return c
}

// Cookies returns all cookies.
func (c *Client) Cookies() []*http.Cookie {
	return c.cookies
}

// SetCookie appends a single cookie.
func (c *Client) SetCookie(hc *http.Cookie) *Client {
	c.cookies = append(c.cookies, hc)
	return c
}

// SetCookies sets multiple cookies.
func (c *Client) SetCookies(cs []*http.Cookie) *Client {
	c.cookies = append(c.cookies, cs...)
	return c
}

// QueryParams returns all query parameters.
func (c *Client) QueryParams() url.Values {
	return c.queryParams
}

// SetQueryParam sets a single parameter and its value.
func (c *Client) SetQueryParam(param, value string) *Client {
	c.queryParams.Set(param, value)
	return c
}

// SetQueryParams sets multiple parameters.
func (c *Client) SetQueryParams(params map[string]string) *Client {
	// Do not lock here since there is potential deadlock.
	for p, v := range params {
		c.SetQueryParam(p, v)
	}
	return c
}

// FormData returns the form parameters.
func (c *Client) FormData() url.Values {
	return c.formData
}

// SetFormData sets form parameters.
func (c *Client) SetFormData(data map[string]string) *Client {
	for k, v := range data {
		c.formData.Set(k, v)
	}
	return c
}

// SetTimeout set timeout for requests fired from the client.
func (c *Client) SetTimeout(d time.Duration) *Client {
	c.httpClient.Timeout = d
	return c
}

// Timeout returns the timeout duration.
func (c *Client) Timeout() time.Duration {
	return c.httpClient.Timeout
}

// R creates a new request.
func (c *Client) R(ctx context.Context) *Request {
	if ctx == nil {
		ctx = context.Background()
	}

	r := &Request{
		QueryParams:        url.Values{},
		FormData:           url.Values{},
		Header:             http.Header{},
		Cookies:            make([]*http.Cookie, 0),
		PathParams:         make(map[string]string),
		Debug:              c.debug,
		IsTrace:            c.isTrace,
		CloseConnection:    c.closeConnection,
		DoNotParseResponse: c.notParseResponse,
		DebugBodyLimit:     c.debugBodyLimit,
		ResponseBodyLimit:  c.responseBodyLimit,

		ctx:              ctx,
		client:           c,
		baseURL:          c.baseURL,
		multipartFields:  make([]*MultipartField, 0),
		setContentLength: c.setContentLength,
	}

	return r
}

// SetRequestMiddlewares sets the request middlewares.
func (c *Client) SetRequestMiddlewares(middlewares ...RequestMiddleware) *Client {
	c.beforeRequest = middlewares
	return c
}

// SetResponseMiddlewares sets the response middlewares.
func (c *Client) SetResponseMiddlewares(middlewares ...ResponseMiddleware) *Client {
	c.afterResponse = middlewares
	return c
}

// AddRequestMiddleware appends a request middleware.
func (c *Client) AddRequestMiddleware(m RequestMiddleware) *Client {
	idx := len(c.beforeRequest) - 1
	c.beforeRequest = slices.Insert(c.beforeRequest, idx, m)
	return c
}

// AddResponseMiddleware appends response middleware.
func (c *Client) AddResponseMiddleware(m ResponseMiddleware) *Client {
	c.afterResponse = append(c.afterResponse, m)
	return c
}

// ContentTypeEncoders returns all registered encoders.
func (c *Client) ContentTypeEncoders() map[string]ContentTypeEncoder {
	return c.contentTypeEncoders
}

// AddContentTypeEncoder adds a Content-Type encoder.
func (c *Client) AddContentTypeEncoder(ct string, e ContentTypeEncoder) *Client {
	c.contentTypeEncoders[ct] = e
	return c
}

func (c *Client) inferContentTypeEncoder(ct ...string) (ContentTypeEncoder, bool) {
	for _, v := range ct {
		if d, f := c.contentTypeEncoders[v]; f {
			return d, f
		}
	}
	return nil, false
}

// ContentTypeDecoders returns all registered decoders.
func (c *Client) ContentTypeDecoders() map[string]ContentTypeDecoder {
	return c.contentTypeDecoders
}

// AddContentTypeDecoder adds a Content-Type decoder.
func (c *Client) AddContentTypeDecoder(ct string, d ContentTypeDecoder) *Client {
	c.contentTypeDecoders[ct] = d
	return c
}

func (c *Client) inferContentTypeDecoder(ct ...string) (ContentTypeDecoder, bool) {
	for _, v := range ct {
		if d, f := c.contentTypeDecoders[v]; f {
			return d, f
		}
	}
	return nil, false
}

// ContentDecompressors returns all registered decompressors.
func (c *Client) ContentDecompressors() map[string]ContentDecompressor {
	return c.contentDecompressors
}

// AddContentDecompressor adds a Content-Encoding decompressor.
func (c *Client) AddContentDecompressor(k string, d ContentDecompressor) *Client {
	if !slices.Contains(c.contentDecompressorKeys, k) {
		c.contentDecompressorKeys = slices.Insert(c.contentDecompressorKeys, 0, k)
	}
	c.contentDecompressors[k] = d
	return c
}

// ContentDecompressorKeys returns decompressor keys.
func (c *Client) ContentDecompressorKeys() string {
	return strings.Join(c.contentDecompressorKeys, ", ")
}

// SetContentDecompressorKeys sets decompressor priority order.
func (c *Client) SetContentDecompressorKeys(keys []string) *Client {
	result := make([]string, 0)
	decoders := c.ContentDecompressors()
	for _, k := range keys {
		if _, f := decoders[k]; f {
			result = append(result, k)
		}
	}

	c.contentDecompressorKeys = result
	return c
}

// IsDebug returns the debug mode status.
func (c *Client) IsDebug() bool {
	return c.debug
}

// SetDebug enables or disables debug mode.
func (c *Client) SetDebug(d bool) *Client {
	c.debug = d
	return c
}

// DebugBodyLimit returns the debug body limit.
func (c *Client) DebugBodyLimit() int {
	return c.debugBodyLimit
}

// SetDebugBodyLimit sets the maximum body size for debug logging.
func (c *Client) SetDebugBodyLimit(sl int) *Client {
	c.debugBodyLimit = sl
	return c
}

// OnDebugLog sets the debug log callback.
func (c *Client) OnDebugLog(dlc DebugLogCallbackFunc) *Client {
	c.debugLogCallback = dlc
	return c
}

// IsContentLength returns the content length status.
func (c *Client) IsContentLength() bool {
	return c.setContentLength
}

// SetContentLength enables the Content-Length header.
func (c *Client) SetContentLength(l bool) *Client {
	c.setContentLength = l
	return c
}

// SetRedirectPolicy sets the redirect policy.
// NOTE: Overwrites previous redirect policies.
func (c *Client) SetRedirectPolicy(policies ...RedirectPolicy) *Client {
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

// Transport returns the underlying http.RoundTripper.
func (c *Client) Transport() http.RoundTripper {
	return c.httpClient.Transport
}

// SetTransport sets a custom http.RoundTripper.
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	if transport == nil {
		panic("SetTransport: transport cannot be nil")
	}
	c.httpClient.Transport = transport
	return c
}

// Scheme returns the custom scheme value.
func (c *Client) Scheme() string {
	return c.scheme
}

// SetScheme sets a custom scheme.
func (c *Client) SetScheme(scheme string) *Client {
	if !isStringEmpty(scheme) {
		c.scheme = strings.TrimSpace(scheme)
	}
	return c
}

// SetCloseConnection sets the Close field in HTTP request.
func (c *Client) SetCloseConnection(close bool) *Client {
	c.closeConnection = close
	return c
}

// SetDoNotParseResponse disables automatic response parsing.
// NOTE: Default response middlewares are not executed.
func (c *Client) SetDoNotParseResponse(notParse bool) *Client {
	c.notParseResponse = notParse
	return c
}

// PathParams returns the path parameters.
func (c *Client) PathParams() map[string]string {
	return c.pathParams
}

// SetPathParam sets a single URL path parameter.
func (c *Client) SetPathParam(param, value string) *Client {
	c.pathParams[param] = value
	return c
}

// SetPathParams sets multiple URL path parameters.
func (c *Client) SetPathParams(params map[string]string) *Client {
	for p, v := range params {
		c.pathParams[p] = v
	}
	return c
}

// ResponseBodyLimit returns the response body size limit.
func (c *Client) ResponseBodyLimit() int64 {
	return c.responseBodyLimit
}

// SetResponseBodyLimit sets the response body size limit.
// NOTE: Limit not enforced when <= 0, or with SetOutputFileName, or DoNotParseResponse.
func (c *Client) SetResponseBodyLimit(v int64) *Client {
	c.responseBodyLimit = v
	return c
}

// SetTrace enables HTTP trace for requests.

// IsTrace returns the trace status.
func (c *Client) IsTrace() bool {
	return c.isTrace
}

// SetTrace enables or disables tracing.
func (c *Client) SetTrace(t bool) *Client {
	c.isTrace = t
	return c
}

// Client returns the underlying http.Client.
func (c *Client) Client() *http.Client {
	return c.httpClient
}

// Clone creates a shallow copy of the client.
// NOTE: Interface values are not deeply cloned. Not safe for concurrent use.
func (c *Client) Clone(ctx context.Context) *Client {
	cc := new(Client)
	// dereference the pointer and copy the value
	*cc = *c

	cc.queryParams = cloneURLValues(c.queryParams)
	cc.formData = cloneURLValues(c.formData)
	cc.header = c.header.Clone()
	cc.pathParams = maps.Clone(c.pathParams)

	cc.contentTypeEncoders = maps.Clone(c.contentTypeEncoders)
	cc.contentTypeDecoders = maps.Clone(c.contentTypeDecoders)
	cc.contentDecompressors = maps.Clone(c.contentDecompressors)
	copy(cc.contentDecompressorKeys, c.contentDecompressorKeys)

	// clone cookies
	if l := len(c.cookies); l > 0 {
		cc.cookies = make([]*http.Cookie, 0, l)
		for _, cookie := range c.cookies {
			cc.cookies = append(cc.cookies, cloneCookie(cookie))
		}
	}

	return cc
}

// Executes method executes the given `Request` object and returns
// response or error.
func (c *Client) execute(req *Request) (*Response, error) {
	for _, f := range c.beforeRequest {
		if err := f(c, req); err != nil {
			return nil, err
		}
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
		if req.Debug {
			response.wrapCopyReadCloser()

			if err = response.readAll(); err != nil {
				return response, err
			}
		}
	}

	debugLogger(c, response)

	// Apply Response middleware
	for _, f := range c.afterResponse {
		if err = f(c, response); err != nil {
			response.Err = wrapErrors(err, response.Err)
		}
	}

	err = response.Err
	return response, err
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
