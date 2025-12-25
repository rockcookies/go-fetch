package fetch

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type (
	// RedirectPolicy regulates client redirects.
	// Apply returns nil to continue, error to stop.
	RedirectPolicy interface {
		Apply(*http.Request, []*http.Request) error
	}

	// RedirectPolicyFunc adapts a function to RedirectPolicy.
	RedirectPolicyFunc func(*http.Request, []*http.Request) error

	// RedirectInfo captures redirect URL and status code.
	RedirectInfo struct {
		URL        string
		StatusCode int
	}
)

// Apply calls f(req, via).
func (f RedirectPolicyFunc) Apply(req *http.Request, via []*http.Request) error {
	return f(req, via)
}

// NoRedirectPolicy disables redirects.
func NoRedirectPolicy() RedirectPolicy {
	return RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})
}

// FlexibleRedirectPolicy creates a redirect policy with max redirects.
func FlexibleRedirectPolicy(noOfRedirect int) RedirectPolicy {
	return RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		if len(via) >= noOfRedirect {
			return fmt.Errorf("resty: stopped after %d redirects", noOfRedirect)
		}
		checkHostAndAddHeaders(req, via[0])
		return nil
	})
}

// DomainCheckRedirectPolicy creates a redirect policy that only allows specified domains.
func DomainCheckRedirectPolicy(hostnames ...string) RedirectPolicy {
	hosts := make(map[string]bool)
	for _, h := range hostnames {
		hosts[strings.ToLower(h)] = true
	}

	return RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		if ok := hosts[getHostname(req.URL.Host)]; !ok {
			return errors.New("redirect is not allowed as per DomainCheckRedirectPolicy")
		}
		checkHostAndAddHeaders(req, via[0])
		return nil
	})
}

func getHostname(host string) (hostname string) {
	if strings.Index(host, ":") > 0 {
		host, _, _ = net.SplitHostPort(host)
	}
	hostname = strings.ToLower(host)
	return
}

// By default, Golang will not redirect request headers.
// After reading through the various discussion comments from the thread -
// https://github.com/golang/go/issues/4800
// Resty will add all the headers during a redirect for the same host and
// adds library user-agent if the Host is different.
func checkHostAndAddHeaders(cur *http.Request, pre *http.Request) {
	curHostname := getHostname(cur.URL.Host)
	preHostname := getHostname(pre.URL.Host)
	if strings.EqualFold(curHostname, preHostname) {
		for key, val := range pre.Header {
			cur.Header[key] = val
		}
	}
}
