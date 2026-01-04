package fetch

import (
	"context"
	"net/http"

	"github.com/rockcookies/go-fetch/internal/utils"
)

// HeaderOptions holds the configuration for HTTP headers in a request.
// The Header field contains all headers that will be set on the request.
type HeaderOptions struct {
	Header http.Header
}

var prepareHeaderKey = utils.NewContextKey[[]func(*HeaderOptions)]("prepare_header")

// PrepareHeaderMiddleware creates a middleware that applies header options from the request context.
// It retrieves header configuration functions stored in the context, executes them to build
// the final HeaderOptions, and replaces the request headers with the configured values.
// This middleware should be used in conjunction with SetHeaderOptions or WithHeaderOptions.
func PrepareHeaderMiddleware() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			options, _ := getOptions(&prepareHeaderKey, req, func() *HeaderOptions {
				return &HeaderOptions{
					Header: req.Header,
				}
			})

			if options == nil {
				return h.Handle(client, req)
			}

			req.Header = options.Header
			return h.Handle(client, req)
		})
	}
}

// SetHeaderOptions creates a middleware that stores header configuration functions in the request.
// These functions will be executed by PrepareHeaderMiddleware to configure headers.
// Multiple configuration functions can be passed and will be applied in sequence.
//
// WithHeaderOptions adds header configuration functions to a context.
// This allows header options to be set at the context level and propagated through the request chain.
// The returned context should be used with http.Request.WithContext.
//
// Example:
//
//	ctx := fetch.WithHeaderOptions(context.Background(), func(opts *fetch.HeaderOptions) {
//	    opts.Header.Set("Authorization", "Bearer token123")
//	})
//	req = req.WithContext(ctx)
//
// Example:
//
//	dispatcher.Use(fetch.SetHeaderOptions(func(opts *fetch.HeaderOptions) {
//	    opts.Header.Set("User-Agent", "MyApp/1.0")
//	    opts.Header.Set("Accept", "application/json")
//	}))
func SetHeaderOptions(opts ...func(*HeaderOptions)) Middleware {
	return withOptionsMiddleware(&prepareHeaderKey, opts...)
}

func WithHeaderOptions(ctx context.Context, opts ...func(*HeaderOptions)) context.Context {
	return withOptions(&prepareHeaderKey, ctx, opts...)
}
