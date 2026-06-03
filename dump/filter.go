package dump

import (
	"net/http"
	"strings"
)

// Filter decides whether a given request/response pair should be dumped.
// Returning false skips all dumping for that call.
type Filter interface {
	Match(req *http.Request, resp *http.Response) bool
}

// FilterFunc adapts a plain function to the Filter interface.
type FilterFunc func(req *http.Request, resp *http.Response) bool

func (f FilterFunc) Match(req *http.Request, resp *http.Response) bool { return f(req, resp) }

// URLFilter matches if the request URL path has any of the given prefixes.
func URLFilter(prefixes ...string) Filter {
	return FilterFunc(func(req *http.Request, _ *http.Response) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(req.URL.Path, p) {
				return true
			}
		}
		return false
	})
}

// MethodFilter matches the given HTTP methods (case-insensitive).
func MethodFilter(methods ...string) Filter {
	set := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		set[strings.ToUpper(m)] = struct{}{}
	}
	return FilterFunc(func(req *http.Request, _ *http.Response) bool {
		_, ok := set[req.Method]
		return ok
	})
}

// StatusFilter matches responses whose status code falls in any [lo, hi] range.
// If resp is nil (network error), no range matches.
func StatusFilter(ranges ...[2]int) Filter {
	return FilterFunc(func(_ *http.Request, resp *http.Response) bool {
		if resp == nil {
			return false
		}
		for _, r := range ranges {
			if resp.StatusCode >= r[0] && resp.StatusCode <= r[1] {
				return true
			}
		}
		return false
	})
}

// HeaderFilter matches if the request contains all specified header keys (non-empty value).
func HeaderFilter(keys ...string) Filter {
	return FilterFunc(func(req *http.Request, _ *http.Response) bool {
		for _, k := range keys {
			if req.Header.Get(k) == "" {
				return false
			}
		}
		return true
	})
}

// AllFilters requires ALL filters to match (AND composition).
func AllFilters(filters ...Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response) bool {
		for _, f := range filters {
			if !f.Match(req, resp) {
				return false
			}
		}
		return true
	})
}

// AnyFilter requires at least one filter to match (OR composition).
func AnyFilter(filters ...Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response) bool {
		for _, f := range filters {
			if f.Match(req, resp) {
				return true
			}
		}
		return false
	})
}

// NotFilter negates a filter.
func NotFilter(f Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response) bool {
		return !f.Match(req, resp)
	})
}

// alwaysFilter is the default filter: dump everything.
var alwaysFilter Filter = FilterFunc(func(_ *http.Request, _ *http.Response) bool { return true })
