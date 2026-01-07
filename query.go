package fetch

import (
	"net/http"
	"net/url"
)

// SetQuery applies functions to modify URL query parameters.
// Functions execute in order. Use this for complex query logic beyond simple key-value pairs.
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

// AddQueryKV adds a query parameter. Preserves existing values for the same key.
func AddQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Add(key, value)
	})
}

// SetQueryKV sets a query parameter. Replaces existing values for the same key.
func SetQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Set(key, value)
	})
}

// AddQueryFromMap adds multiple query parameters from a map. Preserves existing values.
func AddQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Add(k, v)
		}
	})
}

// SetQueryFromMap sets multiple query parameters from a map. Replaces existing values.
func SetQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Set(k, v)
		}
	})
}

// DelQuery removes query parameters by key.
func DelQuery(keys ...string) Middleware {
	return SetQuery(func(query url.Values) {
		for _, k := range keys {
			query.Del(k)
		}
	})
}
