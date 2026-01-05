package fetch

import (
	"net/http"
	"testing"
)

func TestSetHeader(t *testing.T) {
	tests := []struct {
		name          string
		funcs         []func(http.Header)
		expectedKey   string
		expectedValue []string
	}{
		{
			name: "set single header",
			funcs: []func(http.Header){
				func(h http.Header) {
					h.Set("User-Agent", "MyApp/1.0")
				},
			},
			expectedKey:   "User-Agent",
			expectedValue: []string{"MyApp/1.0"},
		},
		{
			name: "set multiple headers",
			funcs: []func(http.Header){
				func(h http.Header) {
					h.Set("User-Agent", "MyApp/1.0")
					h.Set("Accept", "application/json")
				},
			},
			expectedKey:   "Accept",
			expectedValue: []string{"application/json"},
		},
		{
			name: "add multiple values to same header",
			funcs: []func(http.Header){
				func(h http.Header) {
					h.Add("X-Custom", "value1")
					h.Add("X-Custom", "value2")
				},
			},
			expectedKey:   "X-Custom",
			expectedValue: []string{"value1", "value2"},
		},
		{
			name: "multiple functions applied in order",
			funcs: []func(http.Header){
				func(h http.Header) {
					h.Set("Key", "first")
				},
				func(h http.Header) {
					h.Set("Key", "second")
				},
			},
			expectedKey:   "Key",
			expectedValue: []string{"second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetHeader(tt.funcs...)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				values := req.Header[tt.expectedKey]
				if len(values) != len(tt.expectedValue) {
					t.Errorf("expected %d values for %q, got %d", len(tt.expectedValue), tt.expectedKey, len(values))
				}
				for i, v := range tt.expectedValue {
					if values[i] != v {
						t.Errorf("expected value[%d] %q, got %q", i, v, values[i])
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestAddHeaderKV(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         string
		expectedValue string
	}{
		{
			name:          "add simple header",
			key:           "X-Custom",
			value:         "value",
			expectedValue: "value",
		},
		{
			name:          "add user agent",
			key:           "User-Agent",
			value:         "MyApp/1.0",
			expectedValue: "MyApp/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := AddHeaderKV(tt.key, tt.value)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if got := req.Header.Get(tt.key); got != tt.expectedValue {
					t.Errorf("expected %q for key %q, got %q", tt.expectedValue, tt.key, got)
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetHeaderKV(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         string
		expectedValue string
	}{
		{
			name:          "set simple header",
			key:           "Content-Type",
			value:         "application/json",
			expectedValue: "application/json",
		},
		{
			name:          "set authorization header",
			key:           "Authorization",
			value:         "Bearer token",
			expectedValue: "Bearer token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetHeaderKV(tt.key, tt.value)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if got := req.Header.Get(tt.key); got != tt.expectedValue {
					t.Errorf("expected %q for key %q, got %q", tt.expectedValue, tt.key, got)
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestAddHeaderFromMap(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantKeys []string
	}{
		{
			name: "add multiple headers from map",
			headers: map[string]string{
				"X-Custom-1": "value1",
				"X-Custom-2": "value2",
			},
			wantKeys: []string{"X-Custom-1", "X-Custom-2"},
		},
		{
			name:     "empty map does nothing",
			headers:  map[string]string{},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := AddHeaderFromMap(tt.headers)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				for _, key := range tt.wantKeys {
					if got := req.Header.Get(key); got != tt.headers[key] {
						t.Errorf("expected %q for key %q, got %q", tt.headers[key], key, got)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetHeaderFromMap(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantKeys []string
	}{
		{
			name: "set multiple headers from map",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			wantKeys: []string{"Content-Type", "Accept"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetHeaderFromMap(tt.headers)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				for _, key := range tt.wantKeys {
					if got := req.Header.Get(key); got != tt.headers[key] {
						t.Errorf("expected %q for key %q, got %q", tt.headers[key], key, got)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestDelHeader(t *testing.T) {
	tests := []struct {
		name         string
		setupHeaders map[string]string
		keysToDelete []string
		expectedKeys []string
	}{
		{
			name: "delete single header",
			setupHeaders: map[string]string{
				"X-Custom-1": "value1",
				"X-Custom-2": "value2",
			},
			keysToDelete: []string{"X-Custom-1"},
			expectedKeys: []string{"X-Custom-2"},
		},
		{
			name: "delete multiple headers",
			setupHeaders: map[string]string{
				"X-Custom-1": "value1",
				"X-Custom-2": "value2",
				"X-Custom-3": "value3",
			},
			keysToDelete: []string{"X-Custom-1", "X-Custom-3"},
			expectedKeys: []string{"X-Custom-2"},
		},
		{
			name: "delete non-existent header does nothing",
			setupHeaders: map[string]string{
				"X-Custom": "value",
			},
			keysToDelete: []string{"X-Other"},
			expectedKeys: []string{"X-Custom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			// Setup headers
			for k, v := range tt.setupHeaders {
				req.Header.Set(k, v)
			}

			middleware := DelHeader(tt.keysToDelete...)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Check deleted keys are gone
				for _, key := range tt.keysToDelete {
					if _, ok := tt.setupHeaders[key]; ok {
						if req.Header.Get(key) != "" {
							t.Errorf("expected key %q to be deleted", key)
						}
					}
				}

				// Check expected keys still exist
				for _, key := range tt.expectedKeys {
					if got := req.Header.Get(key); got != tt.setupHeaders[key] {
						t.Errorf("expected %q for key %q, got %q", tt.setupHeaders[key], key, got)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "set json content type",
			contentType: "application/json",
		},
		{
			name:        "set xml content type",
			contentType: "application/xml",
		},
		{
			name:        "set plain text content type",
			contentType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetContentType(tt.contentType)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("Content-Type"); got != tt.contentType {
					t.Errorf("expected %q, got %q", tt.contentType, got)
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
	}{
		{
			name:      "set custom user agent",
			userAgent: "MyApp/1.0",
		},
		{
			name:      "set browser user agent",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetUserAgent(tt.userAgent)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("User-Agent"); got != tt.userAgent {
					t.Errorf("expected %q, got %q", tt.userAgent, got)
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}
