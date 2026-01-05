package fetch

import "net/http"

// CookiesAdd returns a middleware that adds one or more HTTP cookies to the outgoing request.
// The cookies are added using the AddCookie method, which properly formats them in the
// Cookie header according to RFC 6265.
//
// Multiple calls to AddCookie with the same cookie name will result in multiple cookies
// being sent. If you need to replace existing cookies, consider using CookiesRemove first.
//
// Example:
//
//	sessionCookie := &http.Cookie{
//	    Name:  "session_id",
//	    Value: "abc123",
//	}
//	authCookie := &http.Cookie{
//	    Name:  "auth_token",
//	    Value: "xyz789",
//	}
//	middleware := fetch.CookiesAdd(sessionCookie, authCookie)
func CookiesAdd(cookies ...*http.Cookie) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, cookie := range cookies {
				req.AddCookie(cookie)
			}
			return h.Handle(client, req)
		})
	}
}

// CookiesRemove returns a middleware that removes all cookies from the outgoing request
// by deleting the Cookie header. This is useful when you need to ensure no cookies are
// sent with a request, regardless of what was set previously.
//
// Note: This removes the entire Cookie header, so all cookies will be removed, not just
// specific ones. If you need selective cookie removal, consider manipulating the Cookie
// header directly using HeaderFuncs.
//
// Example:
//
//	middleware := fetch.CookiesRemove()
func CookiesRemove() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Del("Cookie")
			return h.Handle(client, req)
		})
	}
}
