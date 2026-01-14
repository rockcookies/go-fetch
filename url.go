package fetch

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// SetBaseURL returns a middleware that sets the base URL (scheme and host) for the request.
// If the URI doesn't include a scheme (http:// or https://), it defaults to http://.
//
// This is useful for targeting different environments (dev, staging, prod) or when the
// base URL needs to be determined dynamically.
//
// Example:
//
//	// Both will set the base URL to http://api.example.com
//	fetch.SetBaseURL("http://api.example.com")
//	fetch.SetBaseURL("api.example.com")
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
				// Remove trailing slash to prevent double slashes when concatenating
				basePath := strings.TrimSuffix(u.Path, "/")
				req.URL.Path = basePath + req.URL.Path
			}

			return h.Handle(client, req)
		})
	}
}

// SetPathSuffix returns a middleware that appends a path segment to the request URL's path.
// This is useful for adding API versions or resource identifiers to the end of a path.
//
// Example:
//
//	// Request URL: /api/users
//	// After SetPathSuffix("/123"): /api/users/123
func SetPathSuffix(suffix string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.URL.Path += normalizePath(suffix)
			return h.Handle(client, req)
		})
	}
}

// SetPathPrefix returns a middleware that prepends a path segment to the request URL's path.
// This is useful for adding API base paths or namespace prefixes.
//
// Example:
//
//	// Request URL: /users
//	// After SetPathPrefix("/api/v1"): /api/v1/users
func SetPathPrefix(prefix string) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.URL.Path = normalizePath(prefix) + req.URL.Path
			return h.Handle(client, req)
		})
	}
}

// SetPathParams returns a middleware that replaces path parameter placeholders with actual values.
// Placeholders should be in the format {key}, and they will be replaced with the corresponding
// value from the params map.
//
// This is useful for RESTful APIs with path parameters like /users/{id}/posts/{postId}.
//
// Example:
//
//	// Request URL: /users/{id}/posts/{postId}
//	// After SetPathParams(map[string]string{"id": "123", "postId": "456"})
//	// Result: /users/123/posts/456
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

// normalizePath removes trailing slashes to ensure consistent path handling.
// This prevents double slashes when concatenating path segments.
func normalizePath(path string) string {
	if path == "/" {
		return ""
	}
	return path
}

// normalize ensures the URI has a scheme prefix.
// Defaults to http:// if no scheme is present, simplifying user input.
func normalize(uri string) string {
	match, _ := regexp.MatchString("^http[s]?://", uri)
	if match {
		return uri
	}
	return "http://" + uri
}
