package dump

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Filter determines whether a request/response should be logged.
// Returns true to log, false to skip.
type Filter func(request *http.Request, responseStatus int) bool

// Accept returns the filter as-is for explicit inclusion.
func Accept(filter Filter) Filter { return filter }

// Ignore inverts a filter to exclude matching requests.
func Ignore(filter Filter) Filter {
	return func(r *http.Request, responseStatus int) bool { return !filter(r, responseStatus) }
}

// AcceptMethod returns a filter that accepts requests with specified HTTP methods.
func AcceptMethod(methods ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		reqMethod := strings.ToLower(r.Method)

		for _, method := range methods {
			if strings.ToLower(method) == reqMethod {
				return true
			}
		}

		return false
	}
}

// IgnoreMethod returns a filter that rejects requests with specified HTTP methods.
func IgnoreMethod(methods ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		reqMethod := strings.ToLower(r.Method)

		for _, method := range methods {
			if strings.ToLower(method) == reqMethod {
				return false
			}
		}

		return true
	}
}

// AcceptStatus returns a filter that accepts responses with specified status codes.
func AcceptStatus(statuses ...int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(statuses, responseStatus)
	}
}

// IgnoreStatus returns a filter that rejects responses with specified status codes.
func IgnoreStatus(statuses ...int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(statuses, responseStatus)
	}
}

// AcceptStatusGreaterThan accepts responses with status code greater than the specified value.
func AcceptStatusGreaterThan(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus > status
	}
}

// AcceptStatusGreaterThanOrEqual accepts responses with status code >= the specified value.
func AcceptStatusGreaterThanOrEqual(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus >= status
	}
}

// AcceptStatusLessThan accepts responses with status code less than the specified value.
func AcceptStatusLessThan(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus < status
	}
}

// AcceptStatusLessThanOrEqual accepts responses with status code <= the specified value.
func AcceptStatusLessThanOrEqual(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus <= status
	}
}

// IgnoreStatusGreaterThan rejects responses with status code greater than the specified value.
func IgnoreStatusGreaterThan(status int) Filter {
	return AcceptStatusLessThanOrEqual(status)
}

// IgnoreStatusGreaterThanOrEqual rejects responses with status code >= the specified value.
func IgnoreStatusGreaterThanOrEqual(status int) Filter {
	return AcceptStatusLessThan(status)
}

// IgnoreStatusLessThan rejects responses with status code less than the specified value.
func IgnoreStatusLessThan(status int) Filter {
	return AcceptStatusGreaterThanOrEqual(status)
}

// IgnoreStatusLessThanOrEqual rejects responses with status code <= the specified value.
func IgnoreStatusLessThanOrEqual(status int) Filter {
	return AcceptStatusGreaterThan(status)
}

// AcceptPath accepts requests matching the exact paths specified.
func AcceptPath(urls ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(urls, r.URL.Path)
	}
}

// IgnorePath rejects requests matching the exact paths specified.
func IgnorePath(urls ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(urls, r.URL.Path)
	}
}

// AcceptPathContains accepts requests whose path contains any of the specified strings.
func AcceptPathContains(parts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, part := range parts {
			if strings.Contains(r.URL.Path, part) {
				return true
			}
		}

		return false
	}
}

// IgnorePathContains rejects requests whose path contains any of the specified strings.
func IgnorePathContains(parts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, part := range parts {
			if strings.Contains(r.URL.Path, part) {
				return false
			}
		}

		return true
	}
}

// AcceptPathPrefix accepts requests whose path starts with any of the specified prefixes.
func AcceptPathPrefix(prefixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				return true
			}
		}

		return false
	}
}

// IgnorePathPrefix rejects requests whose path starts with any of the specified prefixes.
func IgnorePathPrefix(prefixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				return false
			}
		}

		return true
	}
}

// AcceptPathSuffix accepts requests whose path ends with any of the specified suffixes.
func AcceptPathSuffix(suffixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(r.URL.Path, suffix) {
				return true
			}
		}

		return false
	}
}

// IgnorePathSuffix rejects requests whose path ends with any of the specified suffixes.
func IgnorePathSuffix(suffixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(r.URL.Path, suffix) {
				return false
			}
		}

		return true
	}
}

// AcceptPathMatch accepts requests whose path matches any of the specified regular expressions.
func AcceptPathMatch(regs ...regexp.Regexp) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, reg := range regs {
			if reg.Match([]byte(r.URL.Path)) {
				return true
			}
		}

		return false
	}
}

// IgnorePathMatch rejects requests whose path matches any of the specified regular expressions.
func IgnorePathMatch(regs ...regexp.Regexp) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, reg := range regs {
			if reg.Match([]byte(r.URL.Path)) {
				return false
			}
		}

		return true
	}
}

// AcceptHost accepts requests matching the exact hosts specified.
func AcceptHost(hosts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(hosts, r.URL.Host)
	}
}

// IgnoreHost rejects requests matching the exact hosts specified.
func IgnoreHost(hosts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(hosts, r.URL.Host)
	}
}

// AcceptHostContains accepts requests whose host contains any of the specified strings.
func AcceptHostContains(parts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, part := range parts {
			if strings.Contains(r.URL.Host, part) {
				return true
			}
		}

		return false
	}
}

// IgnoreHostContains rejects requests whose host contains any of the specified strings.
func IgnoreHostContains(parts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, part := range parts {
			if strings.Contains(r.URL.Host, part) {
				return false
			}
		}

		return true
	}
}

// AcceptHostPrefix accepts requests whose host starts with any of the specified prefixes.
func AcceptHostPrefix(prefixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Host, prefix) {
				return true
			}
		}

		return false
	}
}

// IgnoreHostPrefix rejects requests whose host starts with any of the specified prefixes.
func IgnoreHostPrefix(prefixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Host, prefix) {
				return false
			}
		}

		return true
	}
}

// AcceptHostSuffix accepts requests whose host ends with any of the specified suffixes.
func AcceptHostSuffix(suffixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(r.URL.Host, suffix) {
				return true
			}
		}

		return false
	}
}

// IgnoreHostSuffix rejects requests whose host ends with any of the specified suffixes.
func IgnoreHostSuffix(suffixes ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(r.URL.Host, suffix) {
				return false
			}
		}

		return true
	}
}

// AcceptHostMatch accepts requests whose host matches any of the specified regular expressions.
func AcceptHostMatch(regs ...regexp.Regexp) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, reg := range regs {
			if reg.Match([]byte(r.URL.Host)) {
				return true
			}
		}

		return false
	}
}

// IgnoreHostMatch rejects requests whose host matches any of the specified regular expressions.
func IgnoreHostMatch(regs ...regexp.Regexp) Filter {
	return func(r *http.Request, responseStatus int) bool {
		for _, reg := range regs {
			if reg.Match([]byte(r.URL.Host)) {
				return false
			}
		}

		return true
	}
}
