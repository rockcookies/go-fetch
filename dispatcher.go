// Package fetch provides a flexible and composable HTTP client with middleware support.
package fetch

import (
	"net/http"
	"slices"
)

// Dispatcher manages HTTP client operations with middleware support.
// It wraps an http.Client and applies middleware chains to requests.
// All methods are safe for concurrent use.
type Dispatcher struct {
	client          *http.Client
	middlewares     []Middleware
	coreMiddlewares []Middleware
}

// NewDispatcher creates a new Dispatcher with the given HTTP client and middleware.
// If client is nil, a default client is created with no timeout.
func NewDispatcher(client *http.Client, middlewares ...Middleware) *Dispatcher {
	if client == nil {
		client = &http.Client{
			Transport: http.DefaultTransport,
		}
	}

	return &Dispatcher{
		client:      client,
		middlewares: middlewares,
	}
}

// NewDispatcherWithTransport creates a new Dispatcher with a custom RoundTripper transport.
// A default http.Client with no timeout is created using the provided transport.
func NewDispatcherWithTransport(transport http.RoundTripper, middlewares ...Middleware) *Dispatcher {
	client := &http.Client{
		Transport: transport,
	}

	return &Dispatcher{
		client:      client,
		middlewares: middlewares,
	}
}

// Client returns the underlying HTTP client.
func (d *Dispatcher) Client() *http.Client {
	return d.client
}

// SetClient replaces the underlying HTTP client.
// This operation is safe for concurrent use.
// If client is nil, the method does nothing.
func (d *Dispatcher) SetClient(client *http.Client) {
	if client == nil {
		return
	}

	d.client = client
}

// Middlewares returns the current middleware chain.
func (d *Dispatcher) Middlewares() []Middleware {
	return d.middlewares
}

// SetMiddlewares replaces the current middleware chain.
func (d *Dispatcher) SetMiddlewares(middlewares ...Middleware) {
	d.middlewares = middlewares
}

// Use appends middleware to the dispatcher's middleware chain.
// Note: This modifies the dispatcher in place. If you need an immutable copy,
// use Clone() first.
func (d *Dispatcher) Use(middlewares ...Middleware) {
	d.middlewares = append(d.middlewares, middlewares...)
}

// CoreMiddlewares returns the dispatcher's core middlewares.
func (d *Dispatcher) CoreMiddlewares() []Middleware {
	return d.coreMiddlewares
}

// SetCoreMiddlewares replaces the dispatcher's core middlewares.
func (d *Dispatcher) SetCoreMiddlewares(middlewares ...Middleware) {
	d.coreMiddlewares = middlewares
}

// UseCore appends middleware to the dispatcher's core middleware chain.
// Core middlewares are applied last (innermost layer) in the middleware chain.
// Note: This modifies the dispatcher in place.
func (d *Dispatcher) UseCore(middlewares ...Middleware) {
	d.coreMiddlewares = append(d.coreMiddlewares, middlewares...)
}

// Clone creates a shallow copy of the Dispatcher.
// The HTTP client is cloned, and middlewares are copied.
func (d *Dispatcher) Clone() *Dispatcher {
	return &Dispatcher{
		client:          cloneClient(d.client),
		middlewares:     slices.Clone(d.middlewares),
		coreMiddlewares: slices.Clone(d.coreMiddlewares),
	}
}

// Dispatch executes the HTTP request with the dispatcher's middleware chain.
// Middleware execution order (outermost to innermost):
//  1. d.middlewares (dispatcher's middleware)
//  2. middlewares (per-request middleware)
//  3. d.coreMiddlewares (core middleware, applied last)
func (d *Dispatcher) Dispatch(req *http.Request, middlewares ...Middleware) (*http.Response, error) {
	client := cloneClient(d.client)

	var handler Handler = HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return client.Do(req)
	})

	handler = compose(
		compose(d.middlewares...),
		compose(middlewares...),
		compose(d.coreMiddlewares...),
	)(handler)

	return handler.Handle(client, req)
}

// NewRequest creates a new Request bound to this dispatcher.
func (d *Dispatcher) NewRequest(middlewares ...Middleware) *Request {
	return &Request{
		dispatcher:  d,
		middlewares: middlewares,
	}
}

// R is an alias for NewRequest, creating a new Request bound to this dispatcher.
func (d *Dispatcher) R(middlewares ...Middleware) *Request {
	return d.NewRequest(middlewares...)
}
