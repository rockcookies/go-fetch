// Package fetch provides a flexible and composable HTTP client with middleware support.
package fetch

import (
	"net/http"
	"slices"
	"sync"
	"time"
)

// Dispatcher manages HTTP client operations with middleware support.
// It wraps an http.Client and applies middleware chains to requests.
// All methods are safe for concurrent use.
type Dispatcher struct {
	lock        sync.Mutex
	client      *http.Client
	middlewares []Middleware
}

// NewDispatcher creates a new Dispatcher with the given HTTP client and middleware.
// If client is nil, a default client with 30s timeout is created.
func NewDispatcher(client *http.Client, middlewares ...Middleware) *Dispatcher {
	if client == nil {
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: http.DefaultTransport,
		}
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

	d.lock.Lock()
	defer d.lock.Unlock()

	d.client = client
}

// Middlewares returns the current middleware chain.
func (d *Dispatcher) Middlewares() []Middleware {
	return d.middlewares
}

// Use appends middleware to the dispatcher's middleware chain.
// This operation is safe for concurrent use.
func (d *Dispatcher) Use(middlewares ...Middleware) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.middlewares = append(d.middlewares, middlewares...)
}

// Clone creates a shallow copy of the Dispatcher.
// The HTTP client is cloned, and middlewares are copied.
func (d *Dispatcher) Clone() *Dispatcher {
	return &Dispatcher{
		client:      cloneClient(d.client),
		middlewares: slices.Clone(d.middlewares),
	}
}

// Do executes the HTTP request with the dispatcher's middleware chain
// plus any additional middlewares provided.
func (d *Dispatcher) Do(req *http.Request, middlewares ...Middleware) (*http.Response, error) {
	client := cloneClient(d.client)

	var handler Handler = HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return client.Do(req)
	})

	middlewares = slices.Concat(d.middlewares, middlewares)
	handler = compose(middlewares...)(handler)
	return handler.Handle(client, req)
}

// NewRequest creates a new Request bound to this dispatcher.
func (d *Dispatcher) NewRequest() *Request {
	return &Request{dispatcher: d}
}
