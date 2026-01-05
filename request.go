package fetch

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
)

// Request represents an HTTP request builder that can accumulate middleware
// before being executed. It maintains a reference to its parent Dispatcher
// and builds up a middleware chain.
type Request struct {
	dispatcher  *Dispatcher
	middlewares []Middleware
}

// Use appends middleware to this request's middleware chain.
// Returns the request for method chaining.
func (r *Request) Use(middlewares ...Middleware) *Request {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// Body sets the request body from an io.Reader.
// Options can configure Content-Type and automatic Content-Length.
func (r *Request) Body(reader io.Reader) *Request {
	return r.Use(SetBody(reader))
}

// BodyGet sets the request body using a lazy getter function.
// The function is called when the body is actually needed.
func (r *Request) BodyGet(get func() (io.Reader, error)) *Request {
	return r.Use(SetBodyGet(get))
}

// BodyGetBytes sets the request body using a lazy getter function that returns bytes.
// The function is called when the body is actually needed.
func (r *Request) BodyGetBytes(get func() ([]byte, error)) *Request {
	return r.Use(SetBodyGetBytes(get))
}

// Form sets the request body as URL-encoded form data.
// Automatically sets Content-Type to application/x-www-form-urlencoded.
func (r *Request) Form(form url.Values) *Request {
	return r.Use(SetBodyForm(form))
}

// JSON sets the request body as JSON-encoded data.
// Accepts string, []byte, or any type that can be marshaled to JSON.
// Automatically sets Content-Type to application/json.
func (r *Request) JSON(data any) *Request {
	return r.Use(SetBodyJSON(data))
}

// XML sets the request body as XML-encoded data.
// Accepts string, []byte, or any type that can be marshaled to XML.
// Automatically sets Content-Type to application/xml.
func (r *Request) XML(data any) *Request {
	return r.Use(SetBodyXML(data))
}

func (r *Request) AddCookie(cookie ...*http.Cookie) *Request {
	return r.Use(AddCookie(cookie...))
}

func (r *Request) DelAllCookies() *Request {
	return r.Use(DelAllCookies())
}

func (r *Request) Header(funcs ...func(http.Header)) *Request {
	return r.Use(SetHeader(funcs...))
}

func (r *Request) AddHeaderKV(key, value string) *Request {
	return r.Use(AddHeaderKV(key, value))
}

func (r *Request) HeaderKV(key, value string) *Request {
	return r.Use(SetHeaderKV(key, value))
}

func (r *Request) AddHeaderFromMap(headers map[string]string) *Request {
	return r.Use(AddHeaderFromMap(headers))
}

func (r *Request) HeaderFromMap(headers map[string]string) *Request {
	return r.Use(SetHeaderFromMap(headers))
}

func (r *Request) DelHeader(keys ...string) *Request {
	return r.Use(DelHeader(keys...))
}

func (r *Request) ContentType(contentType string) *Request {
	return r.Use(SetContentType(contentType))
}

func (r *Request) UserAgent(userAgent string) *Request {
	return r.Use(SetUserAgent(userAgent))
}

// Multipart creates a multipart/form-data request body with the given fields.
func (r *Request) Multipart(fields []*MultipartField, opts ...func(*MultipartOptions)) *Request {
	return r.Use(SetMultipart(fields, opts...))
}

func (r *Request) Query(funcs ...func(url.Values)) *Request {
	return r.Use(SetQuery(funcs...))
}

func (r *Request) AddQueryKV(key, value string) *Request {
	return r.Use(AddQueryKV(key, value))
}

func (r *Request) SetQueryKV(key, value string) *Request {
	return r.Use(SetQueryKV(key, value))
}

func (r *Request) AddQueryFromMap(params map[string]string) *Request {
	return r.Use(AddQueryFromMap(params))
}

func (r *Request) SetQueryFromMap(params map[string]string) *Request {
	return r.Use(SetQueryFromMap(params))
}

func (r *Request) DelQuery(keys ...string) *Request {
	return r.Use(DelQuery(keys...))
}

func (r *Request) BaseURL(baseURL string) *Request {
	return r.Use(SetBaseURL(baseURL))
}

func (r *Request) PathSuffix(suffix string) *Request {
	return r.Use(SetPathSuffix(suffix))
}

func (r *Request) PathPrefix(prefix string) *Request {
	return r.Use(SetPathPrefix(prefix))
}

func (r *Request) PathParams(params map[string]string) *Request {
	return r.Use(SetPathParams(params))
}

// Do executes the HTTP request with accumulated middleware.
func (r *Request) Do(req *http.Request) (*http.Response, error) {
	return r.dispatcher.Dispatch(req, r.middlewares...)
}

// Clone creates a shallow copy of the Request.
// The dispatcher reference is preserved, and middleware are copied.
func (r *Request) Clone() *Request {
	return &Request{
		dispatcher:  r.dispatcher,
		middlewares: slices.Clone(r.middlewares),
	}
}

func (r *Request) SendContext(ctx context.Context, method string, u string) *Response {
	if ctx == nil {
		ctx = context.Background()
	}

	req := &http.Request{
		Method:     method,
		URL:        &url.URL{},
		Host:       "",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
	}

	req = req.WithContext(ctx)

	if parsedURL, err := url.Parse(u); err != nil {
		return buildResponse(req, nil, err)
	} else {
		req.URL = parsedURL
	}

	resp, err := r.Do(req)
	return buildResponse(req, resp, err)
}

// Send constructs and executes an HTTP request with the given method and URL.
// Returns a Response which wraps the http.Response or any error.
func (r *Request) Send(method string, u string) *Response {
	return r.SendContext(context.Background(), method, u)
}

// Get method does GET HTTP request. It's defined in section 9.3.1 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.1
func (r *Request) Get(url string) *Response {
	return r.Send("GET", url)
}

func (r *Request) GetContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "GET", url)
}

// Head method does HEAD HTTP request. It's defined in section 9.3.2 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.2
func (r *Request) Head(url string) *Response {
	return r.HeadContext(context.Background(), url)
}

func (r *Request) HeadContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "HEAD", url)
}

// Post method does POST HTTP request. It's defined in section 9.3.3 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.3
func (r *Request) Post(url string) *Response {
	return r.PostContext(context.Background(), url)
}

func (r *Request) PostContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "POST", url)
}

// Put method does PUT HTTP request. It's defined in section 9.3.4 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.4
func (r *Request) Put(url string) *Response {
	return r.PutContext(context.Background(), url)
}

func (r *Request) PutContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "PUT", url)
}

// Patch method does PATCH HTTP request. It's defined in section 2 of [RFC 5789].
//
// [RFC 5789]: https://datatracker.ietf.org/doc/html/rfc5789.html#section-2
func (r *Request) Patch(url string) *Response {
	return r.PatchContext(context.Background(), url)
}

func (r *Request) PatchContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "PATCH", url)
}

// Delete method does DELETE HTTP request. It's defined in section 9.3.5 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.5
func (r *Request) Delete(url string) *Response {
	return r.DeleteContext(context.Background(), url)
}

func (r *Request) DeleteContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "DELETE", url)
}

// Options method does OPTIONS HTTP request. It's defined in section 9.3.7 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.7
func (r *Request) Options(url string) *Response {
	return r.OptionsContext(context.Background(), url)
}

func (r *Request) OptionsContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "OPTIONS", url)
}

// Trace method does TRACE HTTP request. It's defined in section 9.3.8 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.8
func (r *Request) Trace(url string) *Response {
	return r.TraceContext(context.Background(), url)
}

func (r *Request) TraceContext(ctx context.Context, url string) *Response {
	return r.SendContext(ctx, "TRACE", url)
}
