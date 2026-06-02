package fetch

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
)

// RequestFunc modifies or replaces an HTTP request before middleware runs.
// A nil return value means the request is unchanged (in-place mutation is allowed).
type RequestFunc func(req *http.Request) *http.Request

// Apply calls the underlying function.
func (f RequestFunc) Apply(req *http.Request) *http.Request {
	if f == nil {
		return req
	}
	if updated := f(req); updated != nil {
		return updated
	}
	return req
}

// Request represents an HTTP request builder that can accumulate request formatters
// and middleware before being executed. It maintains a reference to its parent
// Dispatcher and builds up a formatter chain and middleware chain.
type Request struct {
	dispatcher  *Dispatcher
	funcs       []RequestFunc
	middlewares []Middleware
}

// Pre prepends middleware to this request's middleware chain.
// Prepended middleware runs before Use-appended middleware.
func (r *Request) Pre(middlewares ...Middleware) *Request {
	r.middlewares = append(middlewares, r.middlewares...)
	return r
}

// Use appends middleware to this request's middleware chain.
// Returns the request for method chaining.
func (r *Request) Use(middlewares ...Middleware) *Request {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// PreFuncs prepends request formatters applied in Do before middleware.
// Prepended formatters run before UseFuncs-appended formatters.
func (r *Request) PreFuncs(funcs ...RequestFunc) *Request {
	r.funcs = append(funcs, r.funcs...)
	return r
}

// UseFuncs appends request formatters applied in Do before middleware.
func (r *Request) UseFuncs(funcs ...RequestFunc) *Request {
	r.funcs = append(r.funcs, funcs...)
	return r
}

// Body sets the request body from an io.Reader.
// Options can configure Content-Type and automatic Content-Length.
func (r *Request) Body(reader io.Reader, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyReader(reader, opts...))
}

// BodyGet sets the request body using a lazy getter function.
// The function is called when the body is actually needed.
func (r *Request) BodyGet(get func() (io.Reader, error), opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyGetReader(get, opts...))
}

// Form sets the request body as URL-encoded form data.
// Automatically sets Content-Type to application/x-www-form-urlencoded.
func (r *Request) Form(form url.Values, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyForm(form, opts...))
}

// JSON sets the request body as JSON-encoded data.
// Accepts string, []byte, or any type that can be marshaled to JSON.
// Automatically sets Content-Type to application/json.
func (r *Request) JSON(data any, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyJSON(data, opts...))
}

// XML sets the request body as XML-encoded data.
// Accepts string, []byte, or any type that can be marshaled to XML.
// Automatically sets Content-Type to application/xml.
func (r *Request) XML(data any, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyXML(data, opts...))
}

// Multipart creates a multipart/form-data request body with the given fields.
func (r *Request) Multipart(fields []*MultipartField, opts ...func(*MultipartOptions)) *Request {
	return r.Use(Multipart(fields, opts...))
}

func (r *Request) applyFuncs(req *http.Request) *http.Request {
	for _, f := range r.funcs {
		req = f.Apply(req)
	}
	return req
}

// Do executes the HTTP request with accumulated formatters and middleware.
func (r *Request) Do(req *http.Request) (*http.Response, error) {
	req = r.applyFuncs(req)
	return r.dispatcher.Do(req, r.middlewares...)
}

// Clone creates a shallow copy of the Request.
// The dispatcher reference is preserved, and formatters and middleware are copied.
func (r *Request) Clone() *Request {
	return &Request{
		dispatcher:  r.dispatcher,
		funcs:       slices.Clone(r.funcs),
		middlewares: slices.Clone(r.middlewares),
	}
}

// Send constructs and executes an HTTP request with the given method and URL.
// Returns a Response which wraps the http.Response or any error.
func (r *Request) Send(ctx context.Context, method string, u string) *Response {
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return buildResponse(nil, nil, err)
	}

	resp, err := r.Do(req)
	return buildResponse(req, resp, err)
}

// Get method does GET HTTP request. It's defined in section 9.3.1 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.1
func (r *Request) Get(ctx context.Context, url string) *Response {
	return r.Send(ctx, "GET", url)
}

// Head method does HEAD HTTP request. It's defined in section 9.3.2 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.2
func (r *Request) Head(ctx context.Context, url string) *Response {
	return r.Send(ctx, "HEAD", url)
}

// Post method does POST HTTP request. It's defined in section 9.3.3 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.3
func (r *Request) Post(ctx context.Context, url string) *Response {
	return r.Send(ctx, "POST", url)
}

// Put method does PUT HTTP request. It's defined in section 9.3.4 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.4
func (r *Request) Put(ctx context.Context, url string) *Response {
	return r.Send(ctx, "PUT", url)
}

// Patch method does PATCH HTTP request. It's defined in section 2 of [RFC 5789].
//
// [RFC 5789]: https://datatracker.ietf.org/doc/html/rfc5789.html#section-2
func (r *Request) Patch(ctx context.Context, url string) *Response {
	return r.Send(ctx, "PATCH", url)
}

// Delete method does DELETE HTTP request. It's defined in section 9.3.5 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.5
func (r *Request) Delete(ctx context.Context, url string) *Response {
	return r.Send(ctx, "DELETE", url)
}

// Options method does OPTIONS HTTP request. It's defined in section 9.3.7 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.7
func (r *Request) Options(ctx context.Context, url string) *Response {
	return r.Send(ctx, "OPTIONS", url)
}

// Trace method does TRACE HTTP request. It's defined in section 9.3.8 of [RFC 9110].
//
// [RFC 9110]: https://datatracker.ietf.org/doc/html/rfc9110.html#section-9.3.8
func (r *Request) Trace(ctx context.Context, url string) *Response {
	return r.Send(ctx, "TRACE", url)
}
