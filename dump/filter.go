package dump

import (
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Filter
// ─────────────────────────────────────────────────────────────────────────────

// Filter decides whether a given exchange should be dumped.
//
// PreFilter: resp and err are always nil.
// PostFilter: resp is nil when a transport error occurred.
type Filter interface {
	Match(req *http.Request, resp *http.Response, err error) bool
}

// FilterFunc adapts a plain function to Filter.
type FilterFunc func(req *http.Request, resp *http.Response, err error) bool

func (f FilterFunc) Match(req *http.Request, resp *http.Response, err error) bool {
	return f(req, resp, err)
}

// AlwaysFilter matches every exchange (used as the default when no filter is set).
var AlwaysFilter Filter = FilterFunc(func(*http.Request, *http.Response, error) bool {
	return true
})

// URLFilter matches when the request path starts with any of the given prefixes.
func URLFilter(prefixes ...string) Filter {
	return FilterFunc(func(req *http.Request, _ *http.Response, _ error) bool {
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
	return FilterFunc(func(req *http.Request, _ *http.Response, _ error) bool {
		_, ok := set[req.Method]
		return ok
	})
}

// StatusFilter matches responses whose status code falls within any [lo, hi] range.
// Never matches when resp is nil (transport error / pre-filter).
func StatusFilter(ranges ...[2]int) Filter {
	return FilterFunc(func(_ *http.Request, resp *http.Response, _ error) bool {
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

// HeaderFilter matches when the request contains all specified header keys.
func HeaderFilter(keys ...string) Filter {
	return FilterFunc(func(req *http.Request, _ *http.Response, _ error) bool {
		for _, k := range keys {
			if req.Header.Get(k) == "" {
				return false
			}
		}
		return true
	})
}

// ErrorFilter matches when a transport-level error occurred.
func ErrorFilter() Filter {
	return FilterFunc(func(_ *http.Request, _ *http.Response, err error) bool {
		return err != nil
	})
}

// AllFilters requires ALL filters to match (AND).
func AllFilters(filters ...Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response, err error) bool {
		for _, f := range filters {
			if !f.Match(req, resp, err) {
				return false
			}
		}
		return true
	})
}

// AnyFilter requires at least one filter to match (OR).
func AnyFilter(filters ...Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response, err error) bool {
		for _, f := range filters {
			if f.Match(req, resp, err) {
				return true
			}
		}
		return false
	})
}

// NotFilter negates a filter.
func NotFilter(f Filter) Filter {
	return FilterFunc(func(req *http.Request, resp *http.Response, err error) bool {
		return !f.Match(req, resp, err)
	})
}
