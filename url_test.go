package fetch

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareURLMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupURL       string
		options        []func(*URLOptions)
		expectedURL    string
		expectedScheme string
		expectedHost   string
		expectedPath   string
	}{
		{
			name:           "base URL only",
			setupURL:       "http://example.com/api",
			options:        []func(*URLOptions){func(o *URLOptions) { o.BaseURL = "https://test.com" }},
			expectedScheme: "https",
			expectedHost:   "test.com",
		},
		{
			name:     "path params replacement",
			setupURL: "http://example.com/users/{id}/posts/{postId}",
			options: []func(*URLOptions){
				func(o *URLOptions) {
					o.PathParams = map[string]string{
						"id":     "123",
						"postId": "456",
					}
				},
			},
			expectedPath: "/users/123/posts/456",
		},
		{
			name:     "query params",
			setupURL: "http://example.com/search",
			options: []func(*URLOptions){
				func(o *URLOptions) {
					o.QueryParams = url.Values{
						"q":     []string{"test"},
						"limit": []string{"10"},
					}
				},
			},
			expectedPath: "/search",
		},
		{
			name:     "combined: base URL, path params, and query params",
			setupURL: "http://localhost/api/users/{id}",
			options: []func(*URLOptions){
				func(o *URLOptions) {
					o.BaseURL = "https://api.example.com"
					o.PathParams = map[string]string{"id": "999"}
					o.QueryParams = url.Values{"include": []string{"profile"}}
				},
			},
			expectedScheme: "https",
			expectedHost:   "api.example.com",
			expectedPath:   "/api/users/999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := PrepareURLMiddleware()

			// Create handler that just returns the modified request
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if tt.expectedScheme != "" {
					assert.Equal(t, tt.expectedScheme, req.URL.Scheme)
				}
				if tt.expectedHost != "" {
					assert.Equal(t, tt.expectedHost, req.URL.Host)
				}
				if tt.expectedPath != "" {
					assert.Equal(t, tt.expectedPath, req.URL.Path)
				}
				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("GET", tt.setupURL, nil)
			require.NoError(t, err)

			// Apply options to context
			ctx := req.Context()
			for _, opt := range tt.options {
				ctx = WithURLOptions(ctx, opt)
			}
			req = req.WithContext(ctx)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
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
			name:     "root path",
			path:     "/",
			expected: "",
		},
		{
			name:     "normal path",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			assert.Equal(t, tt.expected, result)
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
			name:     "http URL",
			uri:      "http://example.com",
			expected: "http://example.com",
		},
		{
			name:     "https URL",
			uri:      "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "domain without protocol",
			uri:      "example.com",
			expected: "http://example.com",
		},
		{
			name:     "localhost without protocol",
			uri:      "localhost:8080",
			expected: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithURLOptions(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		options  []func(*URLOptions)
		validate func(t *testing.T, ctx context.Context)
	}{
		{
			name: "add options to context",
			ctx:  context.Background(),
			options: []func(*URLOptions){
				func(o *URLOptions) { o.BaseURL = "https://api.example.com" },
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
		{
			name: "nil context creates background",
			ctx:  nil,
			options: []func(*URLOptions){
				func(o *URLOptions) { o.BaseURL = "https://test.com" },
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithURLOptions(tt.ctx, tt.options...)
			tt.validate(t, ctx)
		})
	}
}
