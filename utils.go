package fetch

import (
	"net/http"
)

func cloneClient(client *http.Client) *http.Client {
	if client == nil {
		return nil
	}
	clone := *client
	return &clone
}

func applyOptions[T any](options *T, opts ...func(*T)) *T {
	for _, opt := range opts {
		opt(options)
	}

	return options
}

func compose(middlewares ...Middleware) Middleware {
	return func(next Handler) Handler {
		handler := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
