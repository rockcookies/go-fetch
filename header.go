package fetch

import (
	"net/http"
)

// SetHeader applies functions to modify request headers.
// Functions execute in order. Use this for complex header logic beyond simple key-value pairs.
func SetHeader(funcs ...func(h http.Header)) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, f := range funcs {
				f(req.Header)
			}
			return h.Handle(client, req)
		})
	}
}

// AddHeaderKV adds a header value. Preserves existing values for the same key.
func AddHeaderKV(key, value string) Middleware {
	return SetHeader(func(h http.Header) {
		h.Add(key, value)
	})
}

// SetHeaderKV sets a header value. Replaces existing values for the same key.
func SetHeaderKV(key, value string) Middleware {
	return SetHeader(func(h http.Header) {
		h.Set(key, value)
	})
}

// AddHeaderFromMap adds multiple headers from a map. Preserves existing values.
func AddHeaderFromMap(headers map[string]string) Middleware {
	return SetHeader(func(h http.Header) {
		for k, v := range headers {
			h.Add(k, v)
		}
	})
}

// SetHeaderFromMap sets multiple headers from a map. Replaces existing values.
func SetHeaderFromMap(headers map[string]string) Middleware {
	return SetHeader(func(h http.Header) {
		for k, v := range headers {
			h.Set(k, v)
		}
	})
}

// DelHeader removes headers by key.
func DelHeader(keys ...string) Middleware {
	return SetHeader(func(h http.Header) {
		for _, k := range keys {
			h.Del(k)
		}
	})
}

// SetContentType sets the Content-Type header.
func SetContentType(contentType string) Middleware {
	return SetHeader(func(h http.Header) {
		h.Set("Content-Type", contentType)
	})
}

// SetUserAgent sets the User-Agent header.
func SetUserAgent(userAgent string) Middleware {
	return SetHeader(func(h http.Header) {
		h.Set("User-Agent", userAgent)
	})
}
