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

// SetBody returns a middleware that sets the request body from an io.Reader.
// If the reader implements Len() int (like bytes.Buffer), Content-Length is automatically set.
//
// Note: The reader is consumed during the request and cannot be retried.
// For retry support, use SetBodyGet or SetBodyGetBytes instead.
func SetBody(reader io.Reader) Middleware {
	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if reader != nil {
				req.Body = io.NopCloser(reader)

				switch r := reader.(type) {
				case interface{ Len() int }:
					req.ContentLength = int64(r.Len())
				}
			}

			return handler.Handle(client, req)
		})
	}
}

// SetBodyGet creates middleware that lazily provides the request body.
// The getter function is called when the body is needed, supporting retries.
func SetBodyGet(getReader func() (io.Reader, error)) Middleware {
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
			}

			return handler.Handle(client, req)
		})
	}
}

func setBodyGetBytes(getBytes func() ([]byte, error), contentType string) Middleware {
	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if getBytes != nil {
				data, err := getBytes()
				if err != nil {
					return nil, err
				}

				req.ContentLength = int64(len(data))
				req.GetBody = func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(data)), nil
				}

				if contentType != "" {
					req.Header.Set("Content-Type", contentType)
				}
			}

			return handler.Handle(client, req)
		})
	}
}

// SetBodyGetBytes creates middleware that lazily provides the request body as bytes.
// This is more efficient than SetBodyGetBytes when the body size is known.
func SetBodyGetBytes(getBytes func() ([]byte, error)) Middleware {
	return setBodyGetBytes(getBytes, "")
}

// SetBodyJSON creates middleware that marshals data to JSON and sets it as the request body.
// Accepts string, []byte, or any marshallable type.
// Automatically sets Content-Type to application/json.
func SetBodyJSON(data any) Middleware {
	return setBodyGetBytes(func() ([]byte, error) {
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
	}, "application/json")
}

// SetBodyXML creates middleware that marshals data to XML and sets it as the request body.
// Accepts string, []byte, or any marshallable type.
// Automatically sets Content-Type to application/xml.
func SetBodyXML(data any) Middleware {
	return setBodyGetBytes(func() ([]byte, error) {
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
	}, "application/xml")
}

// SetBodyForm creates middleware that encodes form data and sets it as the request body.
// Automatically sets Content-Type to application/x-www-form-urlencoded.
func SetBodyForm(data url.Values) Middleware {
	return setBodyGetBytes(func() ([]byte, error) {
		buf := bufferpool.Get()
		defer bufferpool.Put(buf)

		buf.WriteString(data.Encode())

		return buf.Bytes(), nil
	}, "application/x-www-form-urlencoded")
}
