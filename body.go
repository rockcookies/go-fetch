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

type BodyOptions struct {
	ContentType          string
	AutoSetContentLength bool
}

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
