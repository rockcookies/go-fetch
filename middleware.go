package fetch

import "net/http"

// Handler defines the interface for handling HTTP requests.
// It receives both the HTTP client and request, allowing full control
// over the request execution.
type Handler interface {
	Handle(client *http.Client, req *http.Request) (*http.Response, error)
}

// HandlerFunc is an adapter to allow ordinary functions to be used as Handlers.
type HandlerFunc func(client *http.Client, req *http.Request) (*http.Response, error)

// Handle calls the underlying function.
func (h HandlerFunc) Handle(client *http.Client, req *http.Request) (*http.Response, error) {
	return h(client, req)
}

// Middleware wraps a Handler to add cross-cutting concerns.
// It follows the standard middleware pattern where each middleware
// can decide to call the next handler or short-circuit the chain.
type Middleware func(Handler) Handler

// skip is a no-op middleware that simply passes through to the next handler.
var skip Middleware = func(next Handler) Handler {
	return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return next.Handle(client, req)
	})
}

// Skip returns a no-op middleware that passes requests through unchanged.
func Skip() Middleware {
	return skip
}
