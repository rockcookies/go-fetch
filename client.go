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

	// CloseHook is a callback for client close.
	CloseHook func()

	// RequestFunc manipulates the Request.
	RequestFunc func(*Request) *Request
)

// Client is an HTTP client with configuration.
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

// BaseURL returns the Base URL value.
func (c *Client) BaseURL() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.baseURL
}

// SetBaseURL sets the Base URL.
func (c *Client) SetBaseURL(url string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.baseURL = strings.TrimRight(url, "/")
	return c
}

// Header returns the headers.
func (c *Client) Header() http.Header {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.header
}

// SetHeader sets a single header and its value.
func (c *Client) SetHeader(header, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.header.Set(header, value)
	return c
}

// SetHeaders sets multiple headers and their values.
func (c *Client) SetHeaders(headers map[string]string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	for h, v := range headers {
		c.header.Set(h, v)
	}
	return c
}

// SetHeaderVerbatim sets the header key and value verbatim.
func (c *Client) SetHeaderVerbatim(header, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.header[header] = []string{value}
	return c
}

// Context returns the context.
func (c *Client) Context() context.Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ctx
}

// SetContext sets the context.
func (c *Client) SetContext(ctx context.Context) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ctx = ctx
	return c
}

// CookieJar returns the HTTP cookie jar instance.
func (c *Client) CookieJar() http.CookieJar {
	return c.Client().Jar
}

// SetCookieJar sets a custom cookie jar.
func (c *Client) SetCookieJar(jar http.CookieJar) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.httpClient.Jar = jar
	return c
}

// Cookies returns all cookies.
func (c *Client) Cookies() []*http.Cookie {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cookies
}

// SetCookie appends a single cookie.
func (c *Client) SetCookie(hc *http.Cookie) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cookies = append(c.cookies, hc)
	return c
}

// SetCookies sets multiple cookies.
func (c *Client) SetCookies(cs []*http.Cookie) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cookies = append(c.cookies, cs...)
	return c
}

// QueryParams returns all query parameters.
func (c *Client) QueryParams() url.Values {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.queryParams
}

// SetQueryParam sets a single parameter and its value.
func (c *Client) SetQueryParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
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
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.formData
}

// SetFormData sets form parameters.
func (c *Client) SetFormData(data map[string]string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	for k, v := range data {
		c.formData.Set(k, v)
	}
	return c
}

// R creates a new request.
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
		unescapeQueryParams: c.unescapeQueryParams,
	}

	if c.ctx != nil {
		r.ctx = context.WithoutCancel(c.ctx) // refer to godoc for more info about this function
	}

	return r
}

// NewRequest is an alias for R.
func (c *Client) NewRequest() *Request {
	return c.R()
}

// SetRequestMiddlewares sets the request middlewares.
func (c *Client) SetRequestMiddlewares(middlewares ...RequestMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.beforeRequest = middlewares
	return c
}

// SetResponseMiddlewares sets the response middlewares.
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

// AddRequestMiddleware appends a request middleware.
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

// AddResponseMiddleware appends response middleware.
func (c *Client) AddResponseMiddleware(m ResponseMiddleware) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.afterResponse = append(c.afterResponse, m)
	return c
}

// OnError adds a callback for request failures.
func (c *Client) OnError(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.errorHooks = append(c.errorHooks, h)
	return c
}

// OnSuccess adds a callback for request success.
func (c *Client) OnSuccess(h SuccessHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.successHooks = append(c.successHooks, h)
	return c
}

// OnInvalid adds a callback for pre-execution failures.
func (c *Client) OnInvalid(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.invalidHooks = append(c.invalidHooks, h)
	return c
}

// OnPanic adds a callback for request panics.
func (c *Client) OnPanic(h ErrorHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.panicHooks = append(c.panicHooks, h)
	return c
}

// OnClose adds a callback for client close.
func (c *Client) OnClose(h CloseHook) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.closeHooks = append(c.closeHooks, h)
	return c
}

// ContentTypeEncoders returns all registered encoders.
func (c *Client) ContentTypeEncoders() map[string]ContentTypeEncoder {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentTypeEncoders
}

// AddContentTypeEncoder adds a Content-Type encoder.
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

// ContentTypeDecoders returns all registered decoders.
func (c *Client) ContentTypeDecoders() map[string]ContentTypeDecoder {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentTypeDecoders
}

// AddContentTypeDecoder adds a Content-Type decoder.
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

// ContentDecompressors returns all registered decompressors.
func (c *Client) ContentDecompressors() map[string]ContentDecompressor {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.contentDecompressors
}

// AddContentDecompressor adds a Content-Encoding decompressor.
func (c *Client) AddContentDecompressor(k string, d ContentDecompressor) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !slices.Contains(c.contentDecompressorKeys, k) {
		c.contentDecompressorKeys = slices.Insert(c.contentDecompressorKeys, 0, k)
	}
	c.contentDecompressors[k] = d
	return c
}

// ContentDecompressorKeys returns decompressor keys.
func (c *Client) ContentDecompressorKeys() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
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

	c.lock.Lock()
	defer c.lock.Unlock()
	c.contentDecompressorKeys = result
	return c
}

// IsDebug returns the debug mode status.
func (c *Client) IsDebug() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debug
}

// SetDebug enables or disables debug mode.
func (c *Client) SetDebug(d bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debug = d
	return c
}

// DebugBodyLimit returns the debug body limit.
func (c *Client) DebugBodyLimit() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.debugBodyLimit
}

// SetDebugBodyLimit sets the maximum body size for debug logging.
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

// OnDebugLog sets the debug log callback.
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

// SetDebugLogFormatter sets the debug log formatter.
func (c *Client) SetDebugLogFormatter(df DebugLogFormatterFunc) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.debugLogFormatter = df
	return c
}

// IsDisableWarn returns the warning disable status.
func (c *Client) IsDisableWarn() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.disableWarn
}

// SetDisableWarn enables or disables warning messages.
func (c *Client) SetDisableWarn(d bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.disableWarn = d
	return c
}

// Logger returns the logger instance.
func (c *Client) Logger() Logger {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.log
}

// SetLogger sets the logger instance.
func (c *Client) SetLogger(l Logger) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.log = l
	return c
}

// IsContentLength returns the content length status.
func (c *Client) IsContentLength() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.setContentLength
}

// SetContentLength enables the Content-Length header.
func (c *Client) SetContentLength(l bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.setContentLength = l
	return c
}

// Timeout returns the timeout duration.
func (c *Client) Timeout() time.Duration {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.timeout
}

// SetTimeout sets the timeout for requests.
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.timeout = timeout
	return c
}

// Error returns the error object type.
func (c *Client) Error() reflect.Type {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.errorType
}

// SetError registers the common error object for automatic unmarshalling.
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

// SetRedirectPolicy sets the redirect policy.
// NOTE: Overwrites previous redirect policies.
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

// ProxyURL returns the proxy URL.
func (c *Client) ProxyURL() *url.URL {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.proxyURL
}

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

// OutputDirectory returns the output directory.
func (c *Client) OutputDirectory() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.outputDirectory
}

// SetOutputDirectory sets the output directory for saving HTTP responses.
func (c *Client) SetOutputDirectory(dirPath string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.outputDirectory = dirPath
	return c
}

// HTTPTransport returns the http.Transport.
func (c *Client) HTTPTransport() (*http.Transport, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		return transport, nil
	}
	return nil, ErrNotHttpTransportType
}

// Transport returns the underlying http.RoundTripper.
func (c *Client) Transport() http.RoundTripper {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.httpClient.Transport
}

// SetTransport sets a custom http.RoundTripper.
// NOTE: It overwrites the existing transport.
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if transport != nil {
		c.httpClient.Transport = transport
	}
	return c
}

// Scheme returns the custom scheme value.
func (c *Client) Scheme() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.scheme
}

// SetScheme sets a custom scheme.
func (c *Client) SetScheme(scheme string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !isStringEmpty(scheme) {
		c.scheme = strings.TrimSpace(scheme)
	}
	return c
}

// SetCloseConnection sets the Close field in HTTP request.
func (c *Client) SetCloseConnection(close bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.closeConnection = close
	return c
}

// SetDoNotParseResponse disables automatic response parsing.
// NOTE: Default response middlewares are not executed.
func (c *Client) SetDoNotParseResponse(notParse bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.notParseResponse = notParse
	return c
}

// PathParams returns the path parameters.
func (c *Client) PathParams() map[string]string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.pathParams
}

// SetPathParam sets a single URL path parameter.
// The value will be escaped using url.PathEscape.
func (c *Client) SetPathParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pathParams[param] = url.PathEscape(value)
	return c
}

// SetPathParams sets multiple URL path parameters.
// Values will be escaped using url.PathEscape.
func (c *Client) SetPathParams(params map[string]string) *Client {
	for p, v := range params {
		c.SetPathParam(p, v)
	}
	return c
}

// SetRawPathParam sets a URL path parameter without escaping.
func (c *Client) SetRawPathParam(param, value string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pathParams[param] = value
	return c
}

// SetRawPathParams sets multiple URL path parameters without escaping.
func (c *Client) SetRawPathParams(params map[string]string) *Client {
	for p, v := range params {
		c.SetRawPathParam(p, v)
	}
	return c
}

// SetJSONEscapeHTML enables or disables HTML escape on JSON marshal.
// NOTE: Only applies to standard JSON Marshaller.
func (c *Client) SetJSONEscapeHTML(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.jsonEscapeHTML = b
	return c
}

// ResponseBodyLimit returns the response body size limit.
func (c *Client) ResponseBodyLimit() int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.responseBodyLimit
}

// SetResponseBodyLimit sets the response body size limit.
// NOTE: Limit not enforced when <= 0, or with SetOutputFileName, or DoNotParseResponse.
func (c *Client) SetResponseBodyLimit(v int64) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.responseBodyLimit = v
	return c
}

// SetTrace enables HTTP trace for requests.

// IsTrace returns the trace status.
func (c *Client) IsTrace() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.isTrace
}

// SetTrace enables or disables tracing.
func (c *Client) SetTrace(t bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.isTrace = t
	return c
}

// SetUnescapeQueryParams sets whether to unescape query parameters.
// NOTE: Request failure is possible with non-standard usage.
func (c *Client) SetUnescapeQueryParams(unescape bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.unescapeQueryParams = unescape
	return c
}

// ResponseBodyUnlimitedReads returns the unlimited reads status.
func (c *Client) ResponseBodyUnlimitedReads() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.resBodyUnlimitedReads
}

// SetResponseBodyUnlimitedReads enables unlimited response body reads.
// NOTE: Keeps response body in memory. Also works in debug mode.
func (c *Client) SetResponseBodyUnlimitedReads(b bool) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.resBodyUnlimitedReads = b
	return c
}

// IsProxySet returns whether proxy is configured.
func (c *Client) IsProxySet() bool {
	return c.ProxyURL() != nil
}

// Client returns the underlying http.Client.
func (c *Client) Client() *http.Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.httpClient
}

// Clone creates a shallow copy of the client.
// NOTE: Interface values are not deeply cloned. Not safe for concurrent use.
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

// Close performs cleanup activities.
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
