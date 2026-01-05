package fetch

import "net/http"

// ClientFuncs returns a middleware that applies a series of functions to modify the http.Client
// before making the actual HTTP request. This allows fine-grained control over client-level
// settings such as timeout, redirect policy, and transport configuration.
//
// The functions are executed in the order provided. Each function receives a pointer to the
// http.Client, allowing it to modify the client's properties.
//
// Example:
//
//	middleware := fetch.ClientFuncs(
//	    func(c *http.Client) {
//	        c.Timeout = 10 * time.Second
//	    },
//	    func(c *http.Client) {
//	        c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
//	            return http.ErrUseLastResponse
//	        }
//	    },
//	)
func ClientFuncs(funcs ...func(c *http.Client)) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for _, f := range funcs {
				f(client)
			}
			return h.Handle(client, req)
		})
	}
}
