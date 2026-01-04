package fetch

import (
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

// UseFuncs is a convenience method that wraps functions into middleware.
// Each function receives the http.Request and can modify it before execution.
func (r *Request) UseFuncs(funcs ...func(*http.Request)) *Request {
	return r.Use(func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, f := range funcs {
				f(req)
			}
			return next.Handle(client, req)
		})
	})
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

// Do executes the HTTP request with accumulated middleware.
func (r *Request) Do(req *http.Request) (*http.Response, error) {
	return r.dispatcher.Do(req, r.middlewares...)
}

// Clone creates a shallow copy of the Request.
// The dispatcher reference is preserved, and middleware are copied.
func (r *Request) Clone() *Request {
	return &Request{
		dispatcher:  r.dispatcher,
		middlewares: slices.Clone(r.middlewares),
	}
}

// Send constructs and executes an HTTP request with the given method and URL.
// Returns a Response which wraps the http.Response or any error.
func (r *Request) Send(method string, u string) *Response {
	req := &http.Request{
		Method:     method,
		URL:        &url.URL{},
		Host:       "",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
	}

	if parsedURL, err := url.Parse(u); err != nil {
		return buildResponse(req, nil, err)
	} else {
		req.URL = parsedURL
	}

	resp, err := r.Do(req)
	return buildResponse(req, resp, err)
}
