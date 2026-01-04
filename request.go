package fetch

import (
	"io"
	"net/http"
	"net/url"
	"slices"
)

type Request struct {
	dispatcher  *Dispatcher
	middlewares []Middleware
}

func (r *Request) Use(middlewares ...Middleware) *Request {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

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

func (r *Request) Body(reader io.Reader, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyReader(reader, opts...))
}

func (r *Request) BodyGet(get func() (io.Reader, error), opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyGetReader(get, opts...))
}

func (r *Request) Form(form url.Values, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyForm(form, opts...))
}

func (r *Request) JSON(data any, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyJSON(data, opts...))
}

func (r *Request) XML(data any, opts ...func(*BodyOptions)) *Request {
	return r.Use(BodyXML(data, opts...))
}

func (r *Request) Multipart(fields []*MultipartField, opts ...func(*MultipartOptions)) *Request {
	return r.Use(Multipart(fields, opts...))
}

func (r *Request) Do(req *http.Request) (*http.Response, error) {
	return r.dispatcher.Do(req, r.middlewares...)
}

func (r *Request) Clone() *Request {
	return &Request{
		dispatcher:  r.dispatcher,
		middlewares: slices.Clone(r.middlewares),
	}
}

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
