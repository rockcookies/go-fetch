package fetch

import (
	"context"
	"net/http"

	"github.com/rockcookies/go-fetch/internal/utils"
)

// CookieOptions holds the configuration for HTTP cookies in a request.
// The Cookies field contains all cookies that will be attached to the request.
type CookieOptions struct {
	Cookies []*http.Cookie
}

var prepareCookieKey = utils.NewContextKey[[]func(*CookieOptions)]("prepare_cookie")

// PrepareCookieMiddleware creates a middleware that applies cookie options from the request context.
// It retrieves cookie configuration functions stored in the context, executes them to build
// the final CookieOptions, and attaches all cookies to the outgoing HTTP request.
// This middleware should be used in conjunction with SetCookieOptions or WithCookieOptions.
func PrepareCookieMiddleware() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			options, _ := getOptions(&prepareCookieKey, req, func() *CookieOptions {
				return &CookieOptions{
					Cookies: req.Cookies(),
				}
			})

			if options == nil {
				return h.Handle(client, req)
			}

			req.Header.Del("Cookie")
			for _, cookie := range options.Cookies {
				req.AddCookie(cookie)
			}

			return h.Handle(client, req)
		})
	}
}

// SetCookieOptions creates a middleware that stores cookie configuration functions in the request.
// These functions will be executed by PrepareCookieMiddleware to configure cookies.
// Multiple configuration functions can be passed and will be applied in sequence.
//
// WithCookieOptions adds cookie configuration functions to a context.
// This allows cookie options to be set at the context level and propagated through the request chain.
// The returned context should be used with http.Request.WithContext.
//
// Example:
//
//	ctx := fetch.WithCookieOptions(context.Background(), func(opts *fetch.CookieOptions) {
//	    opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "auth", Value: "secret"})
//	})
//	req = req.WithContext(ctx)
//
// Example:
//
//	dispatcher.Use(fetch.SetCookieOptions(func(opts *fetch.CookieOptions) {
//	    opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "session", Value: "token123"})
//	}))
func SetCookieOptions(opts ...func(*CookieOptions)) Middleware {
	return withOptionsMiddleware(&prepareCookieKey, opts...)
}

func WithCookieOptions(ctx context.Context, opts ...func(*CookieOptions)) context.Context {
	return withOptions(&prepareCookieKey, ctx, opts...)
}
