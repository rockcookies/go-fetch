package dump

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

type Filter func(request *http.Request, responseStatus int) bool

// Basic
func Accept(filter Filter) Filter { return filter }

func Ignore(filter Filter) Filter {
	return func(r *http.Request, responseStatus int) bool { return !filter(r, responseStatus) }
}

// Method
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

// Status
func AcceptStatus(statuses ...int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(statuses, responseStatus)
	}
}

func IgnoreStatus(statuses ...int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(statuses, responseStatus)
	}
}

func AcceptStatusGreaterThan(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus > status
	}
}

func AcceptStatusGreaterThanOrEqual(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus >= status
	}
}

func AcceptStatusLessThan(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus < status
	}
}

func AcceptStatusLessThanOrEqual(status int) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return responseStatus <= status
	}
}

func IgnoreStatusGreaterThan(status int) Filter {
	return AcceptStatusLessThanOrEqual(status)
}

func IgnoreStatusGreaterThanOrEqual(status int) Filter {
	return AcceptStatusLessThan(status)
}

func IgnoreStatusLessThan(status int) Filter {
	return AcceptStatusGreaterThanOrEqual(status)
}

func IgnoreStatusLessThanOrEqual(status int) Filter {
	return AcceptStatusGreaterThan(status)
}

// Path
func AcceptPath(urls ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(urls, r.URL.Path)
	}
}

func IgnorePath(urls ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(urls, r.URL.Path)
	}
}

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

// Host
func AcceptHost(hosts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return slices.Contains(hosts, r.URL.Host)
	}
}

func IgnoreHost(hosts ...string) Filter {
	return func(r *http.Request, responseStatus int) bool {
		return !slices.Contains(hosts, r.URL.Host)
	}
}

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
