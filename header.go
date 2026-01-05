package fetch

import (
	"net/http"
)

// HeaderFuncs returns a middleware that applies a series of functions to modify the request headers
// before making the actual HTTP request. This provides flexible control over header manipulation
// through direct access to the http.Header map.
//
// The functions are executed in the order provided. Each function receives the http.Header from
// the request, allowing it to add, modify, or delete headers as needed.
//
// This is particularly useful for dynamic header manipulation or when you need to perform
// complex header logic that goes beyond simple key-value pairs.
//
// Example:
//
//	middleware := fetch.HeaderFuncs(
//	    func(h http.Header) {
//	        h.Set("User-Agent", "MyApp/1.0")
//	        h.Set("Accept", "application/json")
//	    },
//	    func(h http.Header) {
//	        h.Add("X-Custom-Header", "value1")
//	        h.Add("X-Custom-Header", "value2")  // Multiple values
//	    },
//	)
func HeaderFuncs(funcs ...func(h http.Header)) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, f := range funcs {
				f(req.Header)
			}
			return h.Handle(client, req)
		})
	}
}
