package fetch

import (
	"context"
	"net/http"
	"slices"

	"github.com/rockcookies/go-fetch/internal/utils"
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

func getOptions[T any](key *utils.ContextKey[[]func(T)], req *http.Request, defaults func() T) (T, bool) {
	var options T

	opts, ok := key.GetValue(req.Context())

	if ok {
		options = defaults()

		for _, opt := range opts {
			opt(options)
		}
	}

	return options, ok
}

func withOptions[T any](key *utils.ContextKey[[]T], ctx context.Context, opts ...T) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	existingOpts, _ := key.GetValue(ctx)
	allOpts := append(slices.Clone(existingOpts), opts...)
	return key.WithValue(ctx, allOpts)
}

func withOptionsMiddleware[T any](key *utils.ContextKey[[]T], opts ...T) Middleware {
	if len(opts) == 0 {
		return skip
	}

	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req = req.WithContext(withOptions(key, req.Context(), opts...))
			return h.Handle(client, req)
		})
	}
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
