package fetch

import "net/http"

// AddCookie adds one or more HTTP cookies to the request.
// Multiple cookies with the same name will all be sent.
func AddCookie(cookies ...*http.Cookie) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, cookie := range cookies {
				req.AddCookie(cookie)
			}
			return h.Handle(client, req)
		})
	}
}

// DelAllCookies removes all cookies from the request by deleting the Cookie header.
func DelAllCookies() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Del("Cookie")
			return h.Handle(client, req)
		})
	}
}
