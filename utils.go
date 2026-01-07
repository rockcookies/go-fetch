package fetch

import (
	"net/http"
)

// cloneClient creates a shallow copy of an http.Client.
// This prevents modifications to one client from affecting others.
// Returns nil if the input client is nil.
func cloneClient(client *http.Client) *http.Client {
	if client == nil {
		return nil
	}
	clone := *client
	return &clone
}

// applyOptions applies a series of option functions to an options struct.
// This implements the functional options pattern, which allows for flexible
// and extensible configuration without breaking API compatibility.
func applyOptions[T any](options *T, opts ...func(*T)) *T {
	for _, opt := range opts {
		opt(options)
	}

	return options
}

// compose combines multiple middleware into a single middleware.
// Middleware are applied in the order provided: the first middleware in the slice
// is the outermost layer, and the last middleware is the innermost layer.
//
// This follows the standard middleware composition pattern where each middleware
// wraps the next handler in the chain.
func compose(middlewares ...Middleware) Middleware {
	return func(next Handler) Handler {
		handler := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
