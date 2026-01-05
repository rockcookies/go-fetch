package fetch

import (
	"net/http"
	"net/url"
	"testing"
)

func TestSetQuery(t *testing.T) {
	tests := []struct {
		name        string
		initialURL  string
		funcs       []func(url.Values)
		expectedURL string
	}{
		{
			name:       "add single query parameter",
			initialURL: "http://example.com/path",
			funcs: []func(url.Values){
				func(q url.Values) {
					q.Set("key", "value")
				},
			},
			expectedURL: "http://example.com/path?key=value",
		},
		{
			name:       "add multiple query parameters",
			initialURL: "http://example.com/path",
			funcs: []func(url.Values){
				func(q url.Values) {
					q.Set("key1", "value1")
					q.Set("key2", "value2")
				},
			},
			expectedURL: "http://example.com/path?key1=value1&key2=value2",
		},
		{
			name:       "preserve existing query parameters",
			initialURL: "http://example.com/path?existing=param",
			funcs: []func(url.Values){
				func(q url.Values) {
					q.Set("new", "value")
				},
			},
			expectedURL: "http://example.com/path?existing=param&new=value",
		},
		{
			name:       "multiple functions applied in order",
			initialURL: "http://example.com/path",
			funcs: []func(url.Values){
				func(q url.Values) {
					q.Set("first", "1")
				},
				func(q url.Values) {
					q.Set("second", "2")
				},
			},
			expectedURL: "http://example.com/path?first=1&second=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetQuery(tt.funcs...)
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

func TestAddQueryKV(t *testing.T) {
	tests := []struct {
		name         string
		initialURL   string
		key          string
		value        string
		expectedURL  string
		expectedVals []string
	}{
		{
			name:         "add query parameter to empty URL",
			initialURL:   "http://example.com/path",
			key:          "key",
			value:        "value",
			expectedURL:  "http://example.com/path?key=value",
			expectedVals: []string{"value"},
		},
		{
			name:         "add query parameter with existing params",
			initialURL:   "http://example.com/path?existing=param",
			key:          "new",
			value:        "value",
			expectedURL:  "http://example.com/path?existing=param&new=value",
			expectedVals: []string{"value"},
		},
		{
			name:         "add duplicate key preserves both values",
			initialURL:   "http://example.com/path?key=first",
			key:          "key",
			value:        "second",
			expectedVals: []string{"first", "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := AddQueryKV(tt.key, tt.value)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				vals := req.URL.Query()[tt.key]
				if len(vals) != len(tt.expectedVals) {
					t.Errorf("expected %d values for key %q, got %d", len(tt.expectedVals), tt.key, len(vals))
				}
				for i, v := range tt.expectedVals {
					if vals[i] != v {
						t.Errorf("expected value[%d] %q, got %q", i, v, vals[i])
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetQueryKV(t *testing.T) {
	tests := []struct {
		name        string
		initialURL  string
		key         string
		value       string
		expectedURL string
	}{
		{
			name:        "set query parameter on empty URL",
			initialURL:  "http://example.com/path",
			key:         "key",
			value:       "value",
			expectedURL: "http://example.com/path?key=value",
		},
		{
			name:        "set replaces existing value",
			initialURL:  "http://example.com/path?key=old",
			key:         "key",
			value:       "new",
			expectedURL: "http://example.com/path?key=new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetQueryKV(tt.key, tt.value)
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

func TestAddQueryFromMap(t *testing.T) {
	tests := []struct {
		name       string
		initialURL string
		params     map[string]string
		wantKeys   []string
	}{
		{
			name:       "add multiple params from map",
			initialURL: "http://example.com/path",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantKeys: []string{"key1", "key2"},
		},
		{
			name:       "empty map does nothing",
			initialURL: "http://example.com/path?existing=param",
			params:     map[string]string{},
			wantKeys:   []string{"existing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := AddQueryFromMap(tt.params)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				for _, key := range tt.wantKeys {
					if _, ok := query[key]; !ok {
						t.Errorf("expected key %q to be in query", key)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetQueryFromMap(t *testing.T) {
	tests := []struct {
		name       string
		initialURL string
		params     map[string]string
		wantKeys   []string
	}{
		{
			name:       "set multiple params from map",
			initialURL: "http://example.com/path",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantKeys: []string{"key1", "key2"},
		},
		{
			name:       "set replaces existing values",
			initialURL: "http://example.com/path?key1=old",
			params: map[string]string{
				"key1": "new",
			},
			wantKeys: []string{"key1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetQueryFromMap(tt.params)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				query := req.URL.Query()
				for key, expectedValue := range tt.params {
					if got := query.Get(key); got != expectedValue {
						t.Errorf("expected value %q for key %q, got %q", expectedValue, key, got)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestDelQuery(t *testing.T) {
	tests := []struct {
		name         string
		initialURL   string
		keysToDelete []string
		expectedURL  string
	}{
		{
			name:         "delete single query parameter",
			initialURL:   "http://example.com/path?key1=value1&key2=value2",
			keysToDelete: []string{"key1"},
			expectedURL:  "http://example.com/path?key2=value2",
		},
		{
			name:         "delete multiple query parameters",
			initialURL:   "http://example.com/path?key1=value1&key2=value2&key3=value3",
			keysToDelete: []string{"key1", "key3"},
			expectedURL:  "http://example.com/path?key2=value2",
		},
		{
			name:         "delete all query parameters",
			initialURL:   "http://example.com/path?key1=value1",
			keysToDelete: []string{"key1"},
			expectedURL:  "http://example.com/path",
		},
		{
			name:         "delete non-existent key does nothing",
			initialURL:   "http://example.com/path?key1=value1",
			keysToDelete: []string{"key2"},
			expectedURL:  "http://example.com/path?key1=value1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.initialURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := DelQuery(tt.keysToDelete...)
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
