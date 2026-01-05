package fetch

import (
	"net/http"
	"net/url"
)

// SetQuery returns a middleware that applies a series of functions to modify the request URL query parameters.
// This provides flexible control over query manipulation through direct access to url.Values.
//
// The functions are executed in the order provided. Each function receives the url.Values from
// the request URL, allowing it to add, modify, or delete query parameters.
//
// This is particularly useful for dynamic query manipulation or when you need to perform
// complex query logic that goes beyond simple key-value pairs.
//
// Example:
//
//	middleware := fetch.SetQuery(
//	    func(q url.Values) {
//	        q.Set("api_key", "secret")
//	        q.Set("format", "json")
//	    },
//	    func(q url.Values) {
//	        q.Add("tag", "golang")
//	        q.Add("tag", "http")  // Multiple values
//	    },
//	)
func SetQuery(funcs ...func(query url.Values)) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			query := req.URL.Query()

			for _, f := range funcs {
				f(query)
			}

			req.URL.RawQuery = query.Encode()
			return h.Handle(client, req)
		})
	}
}

// AddQueryKV returns a middleware that adds a single query parameter to the request URL.
// If the key already exists, the value is appended to the existing values (url.Values.Add behavior).
//
// This is useful for adding query parameters that may have multiple values or when you need
// to preserve existing values for the same key.
func AddQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Add(key, value)
	})
}

// SetQueryKV returns a middleware that sets a single query parameter in the request URL.
// If the key already exists, it is replaced with the new value (url.Values.Set behavior).
//
// This is useful for ensuring a query parameter has exactly one value, replacing any existing values.
func SetQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Set(key, value)
	})
}

// AddQueryFromMap returns a middleware that adds multiple query parameters from a map.
// For each map entry, the key-value pair is added to the query parameters.
// Uses url.Values.Add, so existing values are preserved.
func AddQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Add(k, v)
		}
	})
}

// SetQueryFromMap returns a middleware that sets multiple query parameters from a map.
// For each map entry, the key-value pair is set in the query parameters.
// Uses url.Values.Set, so existing values are replaced.
func SetQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Set(k, v)
		}
	})
}

// DelQuery returns a middleware that deletes query parameters by key.
// All values for each specified key are removed from the URL query string.
func DelQuery(keys ...string) Middleware {
	return SetQuery(func(query url.Values) {
		for _, k := range keys {
			query.Del(k)
		}
	})
}
