package fetch

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"

	"github.com/rockcookies/go-fetch/internal/bufferpool"
)

// BodyOptions configures how the request body is handled.
type BodyOptions struct {
	ContentType          string
	AutoSetContentLength bool
}

// BodyReader creates middleware that sets the request body from an io.Reader.
// It can optionally set Content-Type and Content-Length headers.
func BodyReader(reader io.Reader, opts ...func(*BodyOptions)) Middleware {
	options := applyOptions(&BodyOptions{}, opts...)

	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if reader != nil {
				req.Body = io.NopCloser(reader)

				if options.AutoSetContentLength && req.ContentLength == 0 {
					switch r := reader.(type) {
					case interface{ Len() int }:
						req.ContentLength = int64(r.Len())
					}
				}

				if options.ContentType != "" {
					req.Header.Set("Content-Type", options.ContentType)
				}
			}

			return handler.Handle(client, req)
		})
	}
}

// BodyGetReader creates middleware that lazily provides the request body.
// The getter function is called when the body is needed, supporting retries.
func BodyGetReader(getReader func() (io.Reader, error), opts ...func(*BodyOptions)) Middleware {
	options := applyOptions(&BodyOptions{}, opts...)

	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if getReader != nil {
				req.GetBody = func() (io.ReadCloser, error) {
					reader, err := getReader()
					if err != nil {
						return nil, err
					}

					rc, ok := reader.(io.ReadCloser)
					if !ok {
						rc = io.NopCloser(reader)
					}

					return rc, nil
				}

				if options.ContentType != "" {
					req.Header.Set("Content-Type", options.ContentType)
				}
			}

			return handler.Handle(client, req)
		})
	}
}

// BodyGetBytes creates middleware that lazily provides the request body as bytes.
// This is more efficient than BodyGetReader when the body size is known.
func BodyGetBytes(getBytes func() ([]byte, error), opts ...func(*BodyOptions)) Middleware {
	options := applyOptions(&BodyOptions{}, opts...)

	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if getBytes != nil {
				data, err := getBytes()
				if err != nil {
					return nil, err
				}

				req.GetBody = func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(data)), nil
				}

				if options.AutoSetContentLength && req.ContentLength == 0 {
					req.ContentLength = int64(len(data))
				}

				if options.ContentType != "" {
					req.Header.Set("Content-Type", options.ContentType)
				}
			}

			return handler.Handle(client, req)
		})
	}
}

// BodyJSON creates middleware that marshals data to JSON and sets it as the request body.
// Accepts string, []byte, or any marshallable type.
// Automatically sets Content-Type to application/json.
func BodyJSON(data any, opts ...func(*BodyOptions)) Middleware {
	return BodyGetBytes(func() ([]byte, error) {
		switch v := data.(type) {
		case string:
			return []byte(v), nil
		case []byte:
			return v, nil
		default:
			buf := bufferpool.Get()
			defer bufferpool.Put(buf)

			if err := json.NewEncoder(buf).Encode(data); err != nil {
				return nil, err
			}

			return buf.Bytes(), nil
		}
	}, append([]func(*BodyOptions){
		func(o *BodyOptions) {
			o.ContentType = "application/json"
		},
	}, opts...)...)
}

// BodyXML creates middleware that marshals data to XML and sets it as the request body.
// Accepts string, []byte, or any marshallable type.
// Automatically sets Content-Type to application/xml.
func BodyXML(data any, opts ...func(*BodyOptions)) Middleware {
	return BodyGetBytes(func() ([]byte, error) {
		switch v := data.(type) {
		case string:
			return []byte(v), nil
		case []byte:
			return v, nil
		default:
			buf := bufferpool.Get()
			defer bufferpool.Put(buf)

			if err := xml.NewEncoder(buf).Encode(data); err != nil {
				return nil, err
			}

			return buf.Bytes(), nil
		}
	}, append([]func(*BodyOptions){
		func(o *BodyOptions) {
			o.ContentType = "application/xml"
		},
	}, opts...)...)
}

// BodyForm creates middleware that encodes form data and sets it as the request body.
// Automatically sets Content-Type to application/x-www-form-urlencoded.
func BodyForm(data url.Values, opts ...func(*BodyOptions)) Middleware {
	return BodyGetBytes(func() ([]byte, error) {
		buf := bufferpool.Get()
		defer bufferpool.Put(buf)

		buf.WriteString(data.Encode())

		return buf.Bytes(), nil
	}, append([]func(*BodyOptions){
		func(o *BodyOptions) {
			o.ContentType = "application/x-www-form-urlencoded"
		},
	}, opts...)...)
}
