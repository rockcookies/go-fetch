package fetch

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/rockcookies/go-fetch/internal/utils"
)

type URLOptions struct {
	BaseURL     string
	PathParams  map[string]string
	QueryParams url.Values
}

var prepareURLKey = utils.NewContextKey[[]func(*URLOptions)]("prepare_url")

func PrepareURLMiddleware() Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			options, _ := getOptions(&prepareURLKey, req, func() *URLOptions {
				return &URLOptions{
					PathParams:  map[string]string{},
					QueryParams: url.Values{},
				}
			})

			if options == nil {
				return h.Handle(client, req)
			}

			// Apply BaseURL
			if len(options.BaseURL) > 0 {
				baseURL, err := url.Parse(normalize(options.BaseURL))
				if err != nil {
					return nil, &InvalidRequestError{err: err}
				}

				req.URL.Scheme = baseURL.Scheme
				req.URL.Host = baseURL.Host

				if baseURL.Path != "" && baseURL.Path != "/" {
					req.URL.Path = normalizePath(req.URL.Path)
				}
			}

			// Apply PathParams
			for key, value := range options.PathParams {
				placeholder := "{" + key + "}"
				req.URL.Path = strings.ReplaceAll(req.URL.Path, placeholder, value)
			}

			// Apply QueryParams
			if len(options.QueryParams) > 0 {
				if req.URL.RawQuery == "" {
					req.URL.RawQuery = options.QueryParams.Encode()
				} else {
					req.URL.RawQuery = req.URL.RawQuery + "&" + options.QueryParams.Encode()
				}
			}

			return h.Handle(client, req)
		})
	}
}

func SetURLOptions(funcs ...func(*URLOptions)) Middleware {
	return withOptionsMiddleware(&prepareURLKey, funcs...)
}

func WithURLOptions(ctx context.Context, options ...func(*URLOptions)) context.Context {
	return withOptions(&prepareURLKey, ctx, options...)
}

func normalizePath(path string) string {
	if path == "/" {
		return ""
	}
	return path
}

func normalize(uri string) string {
	match, _ := regexp.MatchString("^http[s]?://", uri)
	if match {
		return uri
	}
	return "http://" + uri
}
