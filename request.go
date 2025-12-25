package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Request struct and methods
//_______________________________________________________________________

// Request represents an HTTP request.
type Request struct {
	URL                        string
	Method                     string
	QueryParams                url.Values
	FormData                   url.Values
	PathParams                 map[string]string
	Header                     http.Header
	Time                       time.Time
	Body                       any
	Result                     any
	Error                      any
	RawRequest                 *http.Request
	Cookies                    []*http.Cookie
	Debug                      bool
	CloseConnection            bool
	DoNotParseResponse         bool
	ExpectResponseContentType  string
	ForceResponseContentType   string
	DebugBodyLimit             int
	ResponseBodyLimit          int64
	IsTrace                    bool
	IsDone                     bool
	Timeout                    time.Duration

	isMultiPart       bool
	isFormData        bool
	setContentLength  bool
	jsonEscapeHTML    bool
	ctx               context.Context
	ctxCancelFunc     context.CancelFunc
	values            map[string]any
	client            *Client
	bodyBuf           *bytes.Buffer
	trace             *clientTrace
	log               Logger
	baseURL           string
	multipartBoundary string
	multipartFields   []*MultipartField
	multipartErrChan  chan error
}

// SetMethod sets the HTTP method.
func (r *Request) SetMethod(m string) *Request {
	r.Method = m
	return r
}

// SetURL sets the request URL.
func (r *Request) SetURL(url string) *Request {
	r.URL = url
	return r
}

// Context returns the request's context.
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// SetContext sets the context for the request.
func (r *Request) SetContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// WithContext returns a shallow copy with the context changed.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("resty: Request.WithContext nil context")
	}
	rr := new(Request)
	*rr = *r
	rr.ctx = ctx
	return rr
}

// SetContentType sets the Content-Type header.
func (r *Request) SetContentType(ct string) *Request {
	r.SetHeader(hdrContentTypeKey, ct)
	return r
}

// SetHeader sets a single header field and its value.
func (r *Request) SetHeader(header, value string) *Request {
	r.Header.Set(header, value)
	return r
}

// SetHeaders sets multiple header fields and their values.
func (r *Request) SetHeaders(headers map[string]string) *Request {
	for h, v := range headers {
		r.SetHeader(h, v)
	}
	return r
}

// SetHeaderMultiValues sets multiple header fields with list values.
func (r *Request) SetHeaderMultiValues(headers map[string][]string) *Request {
	for key, values := range headers {
		r.SetHeader(key, strings.Join(values, ", "))
	}
	return r
}

// SetHeaderVerbatim sets the header key and value verbatim.
func (r *Request) SetHeaderVerbatim(header, value string) *Request {
	r.Header[header] = []string{value}
	return r
}

// SetQueryParam sets a single query parameter.
func (r *Request) SetQueryParam(param, value string) *Request {
	r.QueryParams.Set(param, value)
	return r
}

// SetQueryParams sets multiple query parameters.
func (r *Request) SetQueryParams(params map[string]string) *Request {
	for p, v := range params {
		r.SetQueryParam(p, v)
	}
	return r
}

// SetQueryParamsFromValues appends multiple parameters with multi-values.
func (r *Request) SetQueryParamsFromValues(params url.Values) *Request {
	for p, v := range params {
		for _, pv := range v {
			r.QueryParams.Add(p, pv)
		}
	}
	return r
}

// SetQueryString sets the URL query string from a string.
func (r *Request) SetQueryString(query string) *Request {
	params, err := url.ParseQuery(strings.TrimSpace(query))
	if err == nil {
		for p, v := range params {
			for _, pv := range v {
				r.QueryParams.Add(p, pv)
			}
		}
	} else {
		r.log.Errorf("%v", err)
	}
	return r
}

// SetFormData sets form parameters and their values.
func (r *Request) SetFormData(data map[string]string) *Request {
	for k, v := range data {
		r.FormData.Set(k, v)
	}
	return r
}

// SetFormDataFromValues appends form parameters with multi-values.
func (r *Request) SetFormDataFromValues(data url.Values) *Request {
	for k, v := range data {
		for _, kv := range v {
			r.FormData.Add(k, kv)
		}
	}
	return r
}

// SetBody sets the request body.
// Supports string, []byte, struct, map, slice, and io.Reader.
// NOTE: io.Reader is processed in bufferless mode.
func (r *Request) SetBody(body any) *Request {
	r.Body = body
	return r
}

// SetResult registers the response object for automatic unmarshaling.
func (r *Request) SetResult(v any) *Request {
	r.Result = getPointer(v)
	return r
}

// SetError registers the error object for automatic unmarshaling.
func (r *Request) SetError(err any) *Request {
	r.Error = getPointer(err)
	return r
}

// SetFile sets a single file for multipart upload.
func (r *Request) SetFile(fieldName, filePath string) *Request {
	r.isMultiPart = true
	r.multipartFields = append(r.multipartFields, &MultipartField{
		Name:     fieldName,
		FileName: filepath.Base(filePath),
		FilePath: filePath,
	})
	return r
}

// SetFiles sets multiple files for multipart upload.
func (r *Request) SetFiles(files map[string]string) *Request {
	r.isMultiPart = true
	for f, fp := range files {
		r.multipartFields = append(r.multipartFields, &MultipartField{
			Name:     f,
			FileName: filepath.Base(fp),
			FilePath: fp,
		})
	}
	return r
}

// SetFileReader sets a file using io.Reader for multipart upload.
func (r *Request) SetFileReader(fieldName, fileName string, reader io.Reader) *Request {
	r.SetMultipartField(fieldName, fileName, "", reader)
	return r
}

// SetMultipartFormData attaches form data as multipart/form-data.
func (r *Request) SetMultipartFormData(data map[string]string) *Request {
	r.isMultiPart = true
	for k, v := range data {
		r.FormData.Set(k, v)
	}
	return r
}

// SetMultipartOrderedFormData attaches ordered form data as multipart/form-data.
func (r *Request) SetMultipartOrderedFormData(name string, values []string) *Request {
	r.isMultiPart = true
	r.multipartFields = append(r.multipartFields, &MultipartField{
		Name:   name,
		Values: values,
	})
	return r
}

// SetMultipartField sets custom data with Content-Type for multipart upload.
func (r *Request) SetMultipartField(fieldName, fileName, contentType string, reader io.Reader) *Request {
	r.isMultiPart = true
	r.multipartFields = append(r.multipartFields, &MultipartField{
		Name:        fieldName,
		FileName:    fileName,
		ContentType: contentType,
		Reader:      reader,
	})
	return r
}

// SetMultipartFields sets multiple data fields for multipart upload.
func (r *Request) SetMultipartFields(fields ...*MultipartField) *Request {
	r.isMultiPart = true
	r.multipartFields = append(r.multipartFields, fields...)
	return r
}

// SetMultipartBoundary sets the custom multipart boundary.
func (r *Request) SetMultipartBoundary(boundary string) *Request {
	r.multipartBoundary = boundary
	return r
}

// SetContentLength sets the Content-Length header value.
func (r *Request) SetContentLength(l bool) *Request {
	r.setContentLength = l
	return r
}

// SetCloseConnection sets the Close field in HTTP request.
func (r *Request) SetCloseConnection(close bool) *Request {
	r.CloseConnection = close
	return r
}

// SetDoNotParseResponse disables automatic response parsing.
// NOTE: Default response middlewares are not executed.
func (r *Request) SetDoNotParseResponse(notParse bool) *Request {
	r.DoNotParseResponse = notParse
	return r
}

// SetResponseBodyLimit sets the response body size limit.
// NOTE: Limit not enforced when <= 0, or with SetOutputFileName, or DoNotParseResponse.
func (r *Request) SetResponseBodyLimit(v int64) *Request {
	r.ResponseBodyLimit = v
	return r
}


// SetPathParam sets a single URL path parameter (replaces {key} in URL).
func (r *Request) SetPathParam(param, value string) *Request {
	r.PathParams[param] = value
	return r
}

// SetPathParams sets multiple URL path parameters.
func (r *Request) SetPathParams(params map[string]string) *Request {
	for p, v := range params {
		r.PathParams[p] = v
	}
	return r
}

// SetExpectResponseContentType sets the fallback Content-Type for automatic unmarshaling.
func (r *Request) SetExpectResponseContentType(contentType string) *Request {
	r.ExpectResponseContentType = contentType
	return r
}

// SetForceResponseContentType sets a forced Content-Type for automatic unmarshaling.
// Takes priority over the response Content-Type header.
func (r *Request) SetForceResponseContentType(contentType string) *Request {
	r.ForceResponseContentType = contentType
	return r
}

// SetJSONEscapeHTML enables or disables HTML escape on JSON marshal.
// NOTE: Only applies to standard JSON Marshaller.
func (r *Request) SetJSONEscapeHTML(b bool) *Request {
	r.jsonEscapeHTML = b
	return r
}

// SetCookie appends a single cookie.
func (r *Request) SetCookie(hc *http.Cookie) *Request {
	r.Cookies = append(r.Cookies, hc)
	return r
}

// SetCookies sets multiple cookies.
func (r *Request) SetCookies(rs []*http.Cookie) *Request {
	r.Cookies = append(r.Cookies, rs...)
	return r
}

// SetTimeout sets the timeout for the request.
// NOTE: Uses context.WithTimeout, not http.Client.Timeout.
func (r *Request) SetTimeout(timeout time.Duration) *Request {
	r.Timeout = timeout
	return r
}

// SetLogger sets the logger for request and response details.
func (r *Request) SetLogger(l Logger) *Request {
	r.log = l
	return r
}

// SetDebug enables debug mode for logging request and response details.
func (r *Request) SetDebug(d bool) *Request {
	r.Debug = d
	return r
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP request tracing
//_______________________________________________________________________

//

// SetTrace turns on/off the trace capability at the request level.
func (r *Request) SetTrace(t bool) *Request {
	r.IsTrace = t
	return r
}

// TraceInfo returns the trace info for the request.
func (r *Request) TraceInfo() TraceInfo {
	ct := r.trace

	if ct == nil {
		return TraceInfo{}
	}

	ct.lock.RLock()
	defer ct.lock.RUnlock()

	ti := TraceInfo{
		DNSLookup:     0,
		TCPConnTime:   0,
		ServerTime:    0,
		IsConnReused:  ct.gotConnInfo.Reused,
		IsConnWasIdle: ct.gotConnInfo.WasIdle,
		ConnIdleTime:  ct.gotConnInfo.IdleTime,
	}

	if !ct.dnsStart.IsZero() && !ct.dnsDone.IsZero() {
		ti.DNSLookup = ct.dnsDone.Sub(ct.dnsStart)
	}

	if !ct.tlsHandshakeDone.IsZero() && !ct.tlsHandshakeStart.IsZero() {
		ti.TLSHandshake = ct.tlsHandshakeDone.Sub(ct.tlsHandshakeStart)
	}

	if !ct.gotFirstResponseByte.IsZero() && !ct.gotConn.IsZero() {
		ti.ServerTime = ct.gotFirstResponseByte.Sub(ct.gotConn)
	}

	// Calculate the total time accordingly when connection is reused,
	// and DNS start and get conn time may be zero if the request is invalid.
	// See issue #1016.
	requestStartTime := r.Time
	if ct.gotConnInfo.Reused && !ct.getConn.IsZero() {
		requestStartTime = ct.getConn
	} else if !ct.dnsStart.IsZero() {
		requestStartTime = ct.dnsStart
	}
	ti.TotalTime = ct.endTime.Sub(requestStartTime)

	// Only calculate on successful connections
	if !ct.connectDone.IsZero() {
		ti.TCPConnTime = ct.connectDone.Sub(ct.dnsDone)
	}

	// Only calculate on successful connections
	if !ct.gotConn.IsZero() {
		ti.ConnTime = ct.gotConn.Sub(ct.getConn)
	}

	// Only calculate on successful connections
	if !ct.gotFirstResponseByte.IsZero() {
		ti.ResponseTime = ct.endTime.Sub(ct.gotFirstResponseByte)
	}

	// Capture remote address info when connection is non-nil
	if ct.gotConnInfo.Conn != nil {
		ti.RemoteAddr = ct.gotConnInfo.Conn.RemoteAddr().String()
	}

	return ti
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP verb method starts here
//_______________________________________________________________________

// Get performs a GET HTTP request.
func (r *Request) Get(url string) (*Response, error) {
	return r.Execute(MethodGet, url)
}

// Head performs a HEAD HTTP request.
func (r *Request) Head(url string) (*Response, error) {
	return r.Execute(MethodHead, url)
}

// Post performs a POST HTTP request.
func (r *Request) Post(url string) (*Response, error) {
	return r.Execute(MethodPost, url)
}

// Put performs a PUT HTTP request.
func (r *Request) Put(url string) (*Response, error) {
	return r.Execute(MethodPut, url)
}

// Patch performs a PATCH HTTP request.
func (r *Request) Patch(url string) (*Response, error) {
	return r.Execute(MethodPatch, url)
}

// Delete performs a DELETE HTTP request.
func (r *Request) Delete(url string) (*Response, error) {
	return r.Execute(MethodDelete, url)
}

// Options performs an OPTIONS HTTP request.
func (r *Request) Options(url string) (*Response, error) {
	return r.Execute(MethodOptions, url)
}

// Trace performs a TRACE HTTP request.
func (r *Request) Trace(url string) (*Response, error) {
	return r.Execute(MethodTrace, url)
}

// Send performs the HTTP request using the defined method and URL.
func (r *Request) Send() (*Response, error) {
	return r.Execute(r.Method, r.URL)
}

// Execute performs the HTTP request with the given method and URL.
func (r *Request) Execute(method, url string) (res *Response, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if err, ok := rec.(error); ok {
				r.client.onPanicHooks(r, err)
			} else {
				r.client.onPanicHooks(r, fmt.Errorf("panic %v", rec))
			}
			panic(rec)
		}
	}()

	r.Method = method
	r.URL = url

	isInvalidRequestErr := false
	res, err = r.client.execute(r)
	if err != nil {
		if irErr, ok := err.(*invalidRequestError); ok {
			err = irErr.Err
			isInvalidRequestErr = true
		}
	}

	if r.isMultiPart {
		for _, mf := range r.multipartFields {
			mf.close()
		}
	}

	r.IsDone = true

	// Hooks
	if isInvalidRequestErr {
		r.client.onInvalidHooks(r, err)
	} else {
		r.client.onErrorHooks(r, res, err)
	}

	backToBufPool(r.bodyBuf)
	return
}

// Clone returns a deep copy of the request with a new context.
// NOTE: The body is a reference, not a copy.
func (r *Request) Clone(ctx context.Context) *Request {
	if ctx == nil {
		panic("resty: Request.Clone nil context")
	}
	rr := new(Request)
	*rr = *r

	// set new context
	rr.ctx = ctx

	// RawRequest should not copied, since its created on request execution flow.
	rr.RawRequest = nil

	// clone values
	rr.Header = r.Header.Clone()
	rr.FormData = cloneURLValues(r.FormData)
	rr.QueryParams = cloneURLValues(r.QueryParams)
	rr.PathParams = maps.Clone(r.PathParams)

	// clone cookies
	if l := len(r.Cookies); l > 0 {
		rr.Cookies = make([]*http.Cookie, l)
		for _, cookie := range r.Cookies {
			rr.Cookies = append(rr.Cookies, cloneCookie(cookie))
		}
	}

	// create new interface for result and error
	rr.Result = newInterface(r.Result)
	rr.Error = newInterface(r.Error)

	// clone multipart fields
	if l := len(r.multipartFields); l > 0 {
		rr.multipartFields = make([]*MultipartField, l)
		for i, mf := range r.multipartFields {
			rr.multipartFields[i] = mf.Clone()
		}
	}

	// reset values
	rr.Time = time.Time{}
	rr.initTraceIfEnabled()
	r.values = make(map[string]any)
	r.multipartErrChan = nil
	r.ctxCancelFunc = nil

	// copy bodyBuf
	if r.bodyBuf != nil {
		rr.bodyBuf = acquireBuffer()
		rr.bodyBuf.Write(r.bodyBuf.Bytes())
	}

	return rr
}

// Funcs applies RequestFunc functions to the request.
func (r *Request) Funcs(funcs ...RequestFunc) *Request {
	for _, f := range funcs {
		r = f(r)
	}
	return r
}

func (r *Request) fmtBodyString(sl int) (body string) {
	body = "***** NO CONTENT *****"
	if !r.isPayloadSupported() {
		return
	}

	if _, ok := r.Body.(io.Reader); ok {
		body = "***** BODY IS io.Reader *****"
		return
	}

	// multipart or form-data
	if r.isMultiPart || r.isFormData {
		bodySize := r.bodyBuf.Len()
		if bodySize > sl {
			body = fmt.Sprintf("***** REQUEST TOO LARGE (size - %d) *****", bodySize)
			return
		}
		body = r.bodyBuf.String()
		return
	}

	// request body data
	if r.Body == nil {
		return
	}
	var prtBodyBytes []byte
	var err error

	contentType := r.Header.Get(hdrContentTypeKey)
	ctKey := inferContentTypeMapKey(contentType)

	kind := inferKind(r.Body)
	if jsonKey == ctKey &&
		(kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice) {
		buf := acquireBuffer()
		defer releaseBuffer(buf)
		if err = encodeJSONEscapeHTMLIndent(buf, &r.Body, false, "   "); err == nil {
			prtBodyBytes = buf.Bytes()
		}
	} else if xmlKey == ctKey && kind == reflect.Struct {
		prtBodyBytes, err = xml.MarshalIndent(&r.Body, "", "   ")
	} else {
		switch b := r.Body.(type) {
		case string:
			prtBodyBytes = []byte(b)
			if jsonKey == ctKey {
				prtBodyBytes = jsonIndent(prtBodyBytes)
			}
		case []byte:
			body = fmt.Sprintf("***** BODY IS byte(s) (size - %d) *****", len(b))
			return
		}
	}

	bodySize := len(prtBodyBytes)
	if bodySize > sl {
		body = fmt.Sprintf("***** REQUEST TOO LARGE (size - %d) *****", bodySize)
		return
	}

	if prtBodyBytes != nil && err == nil {
		body = string(prtBodyBytes)
	}

	return
}

func (r *Request) initValuesMap() {
	if r.values == nil {
		r.values = make(map[string]any)
	}
}

func (r *Request) initTraceIfEnabled() {
	if r.IsTrace {
		r.trace = new(clientTrace)
		r.ctx = r.trace.createContext(r.Context())
	}
}

func (r *Request) isHeaderExists(k string) bool {
	_, f := r.Header[k]
	return f
}

func (r *Request) writeFormData(w *multipart.Writer) error {
	for k, v := range r.FormData {
		for _, iv := range v {
			if err := w.WriteField(k, iv); err != nil {
				return err
			}
		}
	}
	return nil
}

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

func (r *Request) withTimeout() *http.Request {
	if _, found := r.Context().Deadline(); found {
		return r.RawRequest
	}
	if r.Timeout > 0 {
		ctx, ctxCancelFunc := context.WithTimeout(r.Context(), r.Timeout)
		r.ctxCancelFunc = ctxCancelFunc
		return r.RawRequest.WithContext(ctx)
	}
	return r.RawRequest
}

func jsonIndent(v []byte) []byte {
	buf := acquireBuffer()
	defer releaseBuffer(buf)
	if err := json.Indent(buf, v, "", "   "); err != nil {
		return v
	}
	return buf.Bytes()
}
