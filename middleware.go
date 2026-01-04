package fetch

import "net/http"

type Handler interface {
	Handle(client *http.Client, req *http.Request) (*http.Response, error)
}

type HandlerFunc func(client *http.Client, req *http.Request) (*http.Response, error)

func (h HandlerFunc) Handle(client *http.Client, req *http.Request) (*http.Response, error) {
	return h(client, req)
}

type Middleware func(Handler) Handler

var skip Middleware = func(next Handler) Handler {
	return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return next.Handle(client, req)
	})
}

func Skip() Middleware {
	return skip
}
