package fetch

import (
	"net/http"
	"net/url"
)

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

func AddQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Add(key, value)
	})
}

func SetQueryKV(key, value string) Middleware {
	return SetQuery(func(query url.Values) {
		query.Set(key, value)
	})
}

func AddQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Add(k, v)
		}
	})
}

func SetQueryFromMap(params map[string]string) Middleware {
	return SetQuery(func(query url.Values) {
		for k, v := range params {
			query.Set(k, v)
		}
	})
}

func DelQuery(keys ...string) Middleware {
	return SetQuery(func(query url.Values) {
		for _, k := range keys {
			query.Del(k)
		}
	})
}
