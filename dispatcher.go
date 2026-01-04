package fetch

import (
	"net/http"
	"slices"
	"sync"
	"time"
)

type Dispatcher struct {
	lock        sync.Mutex
	client      *http.Client
	middlewares []Middleware
}

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

func (d *Dispatcher) Client() *http.Client {
	return d.client
}

func (d *Dispatcher) SetClient(client *http.Client) {
	if client == nil {
		return
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	d.client = client
}

func (d *Dispatcher) Middlewares() []Middleware {
	return d.middlewares
}

func (d *Dispatcher) Use(middlewares ...Middleware) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.middlewares = append(d.middlewares, middlewares...)
}

func (d *Dispatcher) Clone() *Dispatcher {
	return &Dispatcher{
		client:      cloneClient(d.client),
		middlewares: slices.Clone(d.middlewares),
	}
}

func (d *Dispatcher) Do(req *http.Request, middlewares ...Middleware) (*http.Response, error) {
	client := cloneClient(d.client)

	var handler Handler = HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return client.Do(req)
	})

	middlewares = slices.Concat(d.middlewares, middlewares)
	handler = compose(middlewares...)(handler)
	return handler.Handle(client, req)
}

func (d *Dispatcher) NewRequest() *Request {
	return &Request{dispatcher: d}
}
