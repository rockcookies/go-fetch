package fetch

import "net/http"

// Handler executes HTTP requests. Receives both client and request to enable
// middleware to modify or replace the client if needed.
type Handler interface {
	Handle(client *http.Client, req *http.Request) (*http.Response, error)
}

// HandlerFunc adapts functions to Handler interface.
type HandlerFunc func(client *http.Client, req *http.Request) (*http.Response, error)

// Handle calls the underlying function.
func (h HandlerFunc) Handle(client *http.Client, req *http.Request) (*http.Response, error) {
	return h(client, req)
}

// Middleware wraps Handler to add cross-cutting concerns.
// Can short-circuit the chain or delegate to next handler.
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
