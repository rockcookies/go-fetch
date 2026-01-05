package fetch

import (
	"net/http"
	"testing"
)

func TestSetBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestURL  string
		expectedURL string
	}{
		{
			name:        "http URL with scheme",
			baseURL:     "http://api.example.com",
			requestURL:  "http://localhost/path",
			expectedURL: "http://api.example.com/path",
		},
		{
			name:        "https URL with scheme",
			baseURL:     "https://api.example.com",
			requestURL:  "http://localhost/path",
			expectedURL: "https://api.example.com/path",
		},
		{
			name:        "URL without scheme defaults to http",
			baseURL:     "api.example.com",
			requestURL:  "http://localhost/path",
			expectedURL: "http://api.example.com/path",
		},
		{
			name:        "base URL with port",
			baseURL:     "http://api.example.com:8080",
			requestURL:  "http://localhost/path",
			expectedURL: "http://api.example.com:8080/path",
		},
		{
			name:        "preserves query parameters",
			baseURL:     "http://api.example.com",
			requestURL:  "http://localhost/path?key=value",
			expectedURL: "http://api.example.com/path?key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.requestURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBaseURL(tt.baseURL)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if req.URL.String() != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, req.URL.String())
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetPathSuffix(t *testing.T) {
	tests := []struct {
		name        string
		initialURL  string
		suffix      string
		expectedURL string
	}{
		{
			name:        "append simple suffix",
			initialURL:  "http://example.com/api",
			suffix:      "/users",
			expectedURL: "http://example.com/api/users",
		},
		{
			name:        "append suffix with leading slash",
			initialURL:  "http://example.com/api/",
			suffix:      "/users",
			expectedURL: "http://example.com/api//users",
		},
		{
			name:        "append suffix without leading slash",
			initialURL:  "http://example.com/api",
			suffix:      "users",
			expectedURL: "http://example.com/apiusers",
		},
		{
			name:        "append numeric suffix",
			initialURL:  "http://example.com/users",
			suffix:      "/123",
			expectedURL: "http://example.com/users/123",
		},
		{
			name:        "preserves query parameters",
			initialURL:  "http://example.com/api?key=value",
			suffix:      "/users",
			expectedURL: "http://example.com/api/users?key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetPathSuffix(tt.suffix)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if req.URL.String() != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, req.URL.String())
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetPathPrefix(t *testing.T) {
	tests := []struct {
		name        string
		initialURL  string
		prefix      string
		expectedURL string
	}{
		{
			name:        "prepend simple prefix",
			initialURL:  "http://example.com/users",
			prefix:      "/api",
			expectedURL: "http://example.com/api/users",
		},
		{
			name:        "prepend API version prefix",
			initialURL:  "http://example.com/users",
			prefix:      "/api/v1",
			expectedURL: "http://example.com/api/v1/users",
		},
		{
			name:        "prefix with trailing slash",
			initialURL:  "http://example.com/users",
			prefix:      "/api/",
			expectedURL: "http://example.com/api//users",
		},
		{
			name:        "preserves query parameters",
			initialURL:  "http://example.com/users?key=value",
			prefix:      "/api",
			expectedURL: "http://example.com/api/users?key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetPathPrefix(tt.prefix)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if req.URL.String() != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, req.URL.String())
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetPathParams(t *testing.T) {
	tests := []struct {
		name        string
		initialURL  string
		params      map[string]string
		expectedURL string
	}{
		{
			name:       "replace single path parameter",
			initialURL: "http://example.com/users/{id}",
			params: map[string]string{
				"id": "123",
			},
			expectedURL: "http://example.com/users/123",
		},
		{
			name:       "replace multiple path parameters",
			initialURL: "http://example.com/users/{userId}/posts/{postId}",
			params: map[string]string{
				"userId": "123",
				"postId": "456",
			},
			expectedURL: "http://example.com/users/123/posts/456",
		},
		{
			name:       "non-existent parameter does nothing",
			initialURL: "http://example.com/users/{id}",
			params: map[string]string{
				"other": "value",
			},
			expectedURL: "http://example.com/users/%7Bid%7D",
		},
		{
			name:       "partial replacement",
			initialURL: "http://example.com/users/{id}/posts/{postId}",
			params: map[string]string{
				"id": "123",
			},
			expectedURL: "http://example.com/users/123/posts/%7BpostId%7D",
		},
		{
			name:       "preserves query parameters",
			initialURL: "http://example.com/users/{id}?key=value",
			params: map[string]string{
				"id": "123",
			},
			expectedURL: "http://example.com/users/123?key=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetPathParams(tt.params)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if req.URL.String() != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, req.URL.String())
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "root path becomes empty",
			path:     "/",
			expected: "",
		},
		{
			name:     "normal path unchanged",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "empty path unchanged",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "http URL unchanged",
			uri:      "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "https URL unchanged",
			uri:      "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "URL without scheme gets http prefix",
			uri:      "example.com",
			expected: "http://example.com",
		},
		{
			name:     "URL with port without scheme gets http prefix",
			uri:      "example.com:8080",
			expected: "http://example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.uri)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
