package dump

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeRequest(method, path, host string) *http.Request {
	return &http.Request{
		Method: method,
		URL: &url.URL{
			Path: path,
			Host: host,
		},
	}
}

func TestAcceptAndIgnore(t *testing.T) {
	alwaysTrue := func(r *http.Request, status int) bool { return true }
	alwaysFalse := func(r *http.Request, status int) bool { return false }

	tests := []struct {
		name     string
		filter   Filter
		expected bool
	}{
		{
			name:     "Accept returns same result",
			filter:   Accept(alwaysTrue),
			expected: true,
		},
		{
			name:     "Ignore inverts result",
			filter:   Ignore(alwaysTrue),
			expected: false,
		},
		{
			name:     "Accept with false",
			filter:   Accept(alwaysFalse),
			expected: false,
		},
		{
			name:     "Ignore with false",
			filter:   Ignore(alwaysFalse),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter(nil, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMethodFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		method   string
		expected bool
	}{
		// AcceptMethod
		{
			name:     "AcceptMethod exact match",
			filter:   AcceptMethod("GET", "POST"),
			method:   "GET",
			expected: true,
		},
		{
			name:     "AcceptMethod case insensitive",
			filter:   AcceptMethod("get"),
			method:   "GET",
			expected: true,
		},
		{
			name:     "AcceptMethod no match",
			filter:   AcceptMethod("GET", "POST"),
			method:   "DELETE",
			expected: false,
		},
		// IgnoreMethod
		{
			name:     "IgnoreMethod exact match",
			filter:   IgnoreMethod("GET", "POST"),
			method:   "GET",
			expected: false,
		},
		{
			name:     "IgnoreMethod no match",
			filter:   IgnoreMethod("GET", "POST"),
			method:   "DELETE",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.method, "/test", "example.com")
			result := tt.filter(req, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		status   int
		expected bool
	}{
		// AcceptStatus
		{
			name:     "AcceptStatus match",
			filter:   AcceptStatus(200, 201, 404),
			status:   200,
			expected: true,
		},
		{
			name:     "AcceptStatus no match",
			filter:   AcceptStatus(200, 201),
			status:   404,
			expected: false,
		},
		// IgnoreStatus
		{
			name:     "IgnoreStatus match",
			filter:   IgnoreStatus(404, 500),
			status:   404,
			expected: false,
		},
		{
			name:     "IgnoreStatus no match",
			filter:   IgnoreStatus(404, 500),
			status:   200,
			expected: true,
		},
		// Greater Than
		{
			name:     "AcceptStatusGreaterThan true",
			filter:   AcceptStatusGreaterThan(400),
			status:   500,
			expected: true,
		},
		{
			name:     "AcceptStatusGreaterThan false",
			filter:   AcceptStatusGreaterThan(400),
			status:   400,
			expected: false,
		},
		// Greater Than Or Equal
		{
			name:     "AcceptStatusGreaterThanOrEqual equal",
			filter:   AcceptStatusGreaterThanOrEqual(400),
			status:   400,
			expected: true,
		},
		{
			name:     "AcceptStatusGreaterThanOrEqual greater",
			filter:   AcceptStatusGreaterThanOrEqual(400),
			status:   500,
			expected: true,
		},
		{
			name:     "AcceptStatusGreaterThanOrEqual less",
			filter:   AcceptStatusGreaterThanOrEqual(400),
			status:   300,
			expected: false,
		},
		// Less Than
		{
			name:     "AcceptStatusLessThan true",
			filter:   AcceptStatusLessThan(400),
			status:   300,
			expected: true,
		},
		{
			name:     "AcceptStatusLessThan false",
			filter:   AcceptStatusLessThan(400),
			status:   400,
			expected: false,
		},
		// Less Than Or Equal
		{
			name:     "AcceptStatusLessThanOrEqual equal",
			filter:   AcceptStatusLessThanOrEqual(400),
			status:   400,
			expected: true,
		},
		{
			name:     "AcceptStatusLessThanOrEqual less",
			filter:   AcceptStatusLessThanOrEqual(400),
			status:   300,
			expected: true,
		},
		{
			name:     "AcceptStatusLessThanOrEqual greater",
			filter:   AcceptStatusLessThanOrEqual(400),
			status:   500,
			expected: false,
		},
		// Ignore variants
		{
			name:     "IgnoreStatusGreaterThan",
			filter:   IgnoreStatusGreaterThan(400),
			status:   500,
			expected: false,
		},
		{
			name:     "IgnoreStatusLessThan",
			filter:   IgnoreStatusLessThan(400),
			status:   300,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest("GET", "/test", "example.com")
			result := tt.filter(req, tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		path     string
		expected bool
	}{
		// AcceptPath
		{
			name:     "AcceptPath exact match",
			filter:   AcceptPath("/api/users", "/api/posts"),
			path:     "/api/users",
			expected: true,
		},
		{
			name:     "AcceptPath no match",
			filter:   AcceptPath("/api/users", "/api/posts"),
			path:     "/api/comments",
			expected: false,
		},
		// IgnorePath
		{
			name:     "IgnorePath match",
			filter:   IgnorePath("/health", "/metrics"),
			path:     "/health",
			expected: false,
		},
		{
			name:     "IgnorePath no match",
			filter:   IgnorePath("/health", "/metrics"),
			path:     "/api/users",
			expected: true,
		},
		// Contains
		{
			name:     "AcceptPathContains match",
			filter:   AcceptPathContains("admin", "internal"),
			path:     "/api/admin/users",
			expected: true,
		},
		{
			name:     "AcceptPathContains no match",
			filter:   AcceptPathContains("admin", "internal"),
			path:     "/api/users",
			expected: false,
		},
		{
			name:     "IgnorePathContains match",
			filter:   IgnorePathContains("test", "debug"),
			path:     "/api/test/endpoint",
			expected: false,
		},
		{
			name:     "IgnorePathContains no match",
			filter:   IgnorePathContains("test", "debug"),
			path:     "/api/users",
			expected: true,
		},
		// Prefix
		{
			name:     "AcceptPathPrefix match",
			filter:   AcceptPathPrefix("/api/v1", "/api/v2"),
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name:     "AcceptPathPrefix no match",
			filter:   AcceptPathPrefix("/api/v1", "/api/v2"),
			path:     "/users",
			expected: false,
		},
		{
			name:     "IgnorePathPrefix match",
			filter:   IgnorePathPrefix("/internal", "/private"),
			path:     "/internal/config",
			expected: false,
		},
		{
			name:     "IgnorePathPrefix no match",
			filter:   IgnorePathPrefix("/internal", "/private"),
			path:     "/api/users",
			expected: true,
		},
		// Suffix
		{
			name:     "AcceptPathSuffix match",
			filter:   AcceptPathSuffix(".json", ".xml"),
			path:     "/api/users.json",
			expected: true,
		},
		{
			name:     "AcceptPathSuffix no match",
			filter:   AcceptPathSuffix(".json", ".xml"),
			path:     "/api/users",
			expected: false,
		},
		{
			name:     "IgnorePathSuffix match",
			filter:   IgnorePathSuffix(".html", ".txt"),
			path:     "/doc/readme.html",
			expected: false,
		},
		{
			name:     "IgnorePathSuffix no match",
			filter:   IgnorePathSuffix(".html", ".txt"),
			path:     "/api/users.json",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest("GET", tt.path, "example.com")
			result := tt.filter(req, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathMatchFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		path     string
		expected bool
	}{
		{
			name:     "AcceptPathMatch matches",
			filter:   AcceptPathMatch(*regexp.MustCompile(`^/api/v\d+/`), *regexp.MustCompile(`/users$`)),
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name:     "AcceptPathMatch no match",
			filter:   AcceptPathMatch(*regexp.MustCompile(`^/api/v\d+/`)),
			path:     "/users",
			expected: false,
		},
		{
			name:     "IgnorePathMatch matches",
			filter:   IgnorePathMatch(*regexp.MustCompile(`/test/`)),
			path:     "/api/test/endpoint",
			expected: false,
		},
		{
			name:     "IgnorePathMatch no match",
			filter:   IgnorePathMatch(*regexp.MustCompile(`/test/`)),
			path:     "/api/users",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest("GET", tt.path, "example.com")
			result := tt.filter(req, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHostFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		host     string
		expected bool
	}{
		// AcceptHost
		{
			name:     "AcceptHost exact match",
			filter:   AcceptHost("api.example.com", "www.example.com"),
			host:     "api.example.com",
			expected: true,
		},
		{
			name:     "AcceptHost no match",
			filter:   AcceptHost("api.example.com", "www.example.com"),
			host:     "other.example.com",
			expected: false,
		},
		// IgnoreHost
		{
			name:     "IgnoreHost match",
			filter:   IgnoreHost("localhost", "127.0.0.1"),
			host:     "localhost",
			expected: false,
		},
		{
			name:     "IgnoreHost no match",
			filter:   IgnoreHost("localhost", "127.0.0.1"),
			host:     "example.com",
			expected: true,
		},
		// Contains
		{
			name:     "AcceptHostContains match",
			filter:   AcceptHostContains("staging", "dev"),
			host:     "api.staging.example.com",
			expected: true,
		},
		{
			name:     "AcceptHostContains no match",
			filter:   AcceptHostContains("staging", "dev"),
			host:     "api.example.com",
			expected: false,
		},
		{
			name:     "IgnoreHostContains match",
			filter:   IgnoreHostContains("test", "localhost"),
			host:     "test.example.com",
			expected: false,
		},
		{
			name:     "IgnoreHostContains no match",
			filter:   IgnoreHostContains("test", "localhost"),
			host:     "api.example.com",
			expected: true,
		},
		// Prefix
		{
			name:     "AcceptHostPrefix match",
			filter:   AcceptHostPrefix("api.", "www."),
			host:     "api.example.com",
			expected: true,
		},
		{
			name:     "AcceptHostPrefix no match",
			filter:   AcceptHostPrefix("api.", "www."),
			host:     "mail.example.com",
			expected: false,
		},
		{
			name:     "IgnoreHostPrefix match",
			filter:   IgnoreHostPrefix("internal.", "private."),
			host:     "internal.example.com",
			expected: false,
		},
		{
			name:     "IgnoreHostPrefix no match",
			filter:   IgnoreHostPrefix("internal.", "private."),
			host:     "api.example.com",
			expected: true,
		},
		// Suffix
		{
			name:     "AcceptHostSuffix match",
			filter:   AcceptHostSuffix(".com", ".org"),
			host:     "example.com",
			expected: true,
		},
		{
			name:     "AcceptHostSuffix no match",
			filter:   AcceptHostSuffix(".com", ".org"),
			host:     "example.net",
			expected: false,
		},
		{
			name:     "IgnoreHostSuffix match",
			filter:   IgnoreHostSuffix(".local", ".internal"),
			host:     "service.local",
			expected: false,
		},
		{
			name:     "IgnoreHostSuffix no match",
			filter:   IgnoreHostSuffix(".local", ".internal"),
			host:     "example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest("GET", "/test", tt.host)
			result := tt.filter(req, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHostMatchFilters(t *testing.T) {
	tests := []struct {
		name     string
		filter   Filter
		host     string
		expected bool
	}{
		{
			name:     "AcceptHostMatch matches",
			filter:   AcceptHostMatch(*regexp.MustCompile(`^api\.`), *regexp.MustCompile(`\.com$`)),
			host:     "api.example.com",
			expected: true,
		},
		{
			name:     "AcceptHostMatch no match",
			filter:   AcceptHostMatch(*regexp.MustCompile(`^api\.`)),
			host:     "www.example.com",
			expected: false,
		},
		{
			name:     "IgnoreHostMatch matches",
			filter:   IgnoreHostMatch(*regexp.MustCompile(`localhost`)),
			host:     "localhost",
			expected: false,
		},
		{
			name:     "IgnoreHostMatch no match",
			filter:   IgnoreHostMatch(*regexp.MustCompile(`localhost`)),
			host:     "example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest("GET", "/test", tt.host)
			result := tt.filter(req, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterCombinations(t *testing.T) {
	// Test combining multiple filters using Accept and Ignore
	methodFilter := AcceptMethod("POST", "PUT")
	statusFilter := AcceptStatusGreaterThanOrEqual(200)
	pathFilter := AcceptPathPrefix("/api/")

	req := makeRequest("POST", "/api/users", "example.com")

	assert.True(t, methodFilter(req, 200))
	assert.True(t, statusFilter(req, 200))
	assert.True(t, pathFilter(req, 200))

	// Test with non-matching
	req2 := makeRequest("GET", "/public/users", "example.com")
	assert.False(t, methodFilter(req2, 200))
	assert.False(t, pathFilter(req2, 200))
}
