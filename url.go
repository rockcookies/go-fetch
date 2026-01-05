package fetch

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func SetBaseURL(uri string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			u, err := url.Parse(normalize(uri))
			if err != nil {
				return nil, err
			}

			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host

			if u.Path != "" && u.Path != "/" {
				req.URL.Path = normalizePath(req.URL.Path)
			}

			return h.Handle(client, req)
		})
	}
}

func SetPathSuffix(suffix string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.URL.Path += normalizePath(suffix)
			return h.Handle(client, req)
		})
	}
}

func SetPathPrefix(prefix string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.URL.Path = normalizePath(prefix) + req.URL.Path
			return h.Handle(client, req)
		})
	}
}

func SetPathParams(params map[string]string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			for key, value := range params {
				placeholder := "{" + key + "}"
				req.URL.Path = strings.ReplaceAll(req.URL.Path, placeholder, value)
			}
			return h.Handle(client, req)
		})
	}
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
