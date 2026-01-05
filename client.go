package fetch

import (
	"context"
	"net/http"

	"github.com/rockcookies/go-fetch/internal/utils"
)

var prepareClientKey = utils.NewContextKey[[]func(*http.Client)]("prepare_client")

// PrepareClientMiddleware returns a middleware that applies client configuration
// options stored in the request context. This middleware retrieves client options
// from the context (set via WithClientOptions) and applies them to create a
// configured http.Client before passing it to the next handler.
//
// The middleware allows for dynamic client configuration on a per-request basis,
// enabling features like custom timeouts, transport settings, and redirect policies.
//
// Example:
//
//	dispatcher := NewDispatcher(nil)
//	req := dispatcher.NewRequest().Use(PrepareClientMiddleware())
//	ctx := WithClientOptions(context.Background(), func(c *http.Client) {
//	    c.Timeout = 30 * time.Second
//	})
//	resp := req.SendWithContext(ctx, "GET", "https://api.example.com")
func PrepareClientMiddleware() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			options, _ := getOptions(&prepareClientKey, req, func() *http.Client {
				return client
			})

			if options != nil {
				client = options
			}

			return h.Handle(client, req)
		})
	}
}

// SetClientOptions returns a middleware that configures the http.Client with
// the provided option functions. Each option function receives a pointer to
// the http.Client and can modify its properties such as Timeout, Transport,
// CheckRedirect, and other fields.
//
// Options are applied in the order they are provided. This middleware is typically
// used with PrepareClientMiddleware to enable client customization.
//
// Example:
//
//	req := dispatcher.NewRequest().Use(
//	    PrepareClientMiddleware(),
//	    SetClientOptions(
//	        func(c *http.Client) { c.Timeout = 10 * time.Second },
//	        func(c *http.Client) {
//	            c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
//	                return http.ErrUseLastResponse
//	            }
//	        },
//	    ),
//	)
func SetClientOptions(opts ...func(*http.Client)) Middleware {
	return withOptionsMiddleware(&prepareClientKey, opts...)
}

// WithClientOptions adds http.Client configuration functions to the context.
// These options will be applied by PrepareClientMiddleware when the request
// is executed. This function allows for context-based client configuration,
// which is useful for setting client properties dynamically based on runtime
// conditions.
//
// Multiple calls to WithClientOptions on the same context will accumulate options.
// All options will be applied in the order they were added.
//
// Example:
//
//	ctx := context.Background()
//	ctx = WithClientOptions(ctx, func(c *http.Client) {
//	    c.Timeout = 30 * time.Second
//	})
//	ctx = WithClientOptions(ctx, func(c *http.Client) {
//	    c.Transport = &http.Transport{MaxIdleConns: 10}
//	})
//	resp := req.SendWithContext(ctx, "GET", "https://api.example.com")
func WithClientOptions(ctx context.Context, opts ...func(*http.Client)) context.Context {
	return withOptions(&prepareClientKey, ctx, opts...)
}
