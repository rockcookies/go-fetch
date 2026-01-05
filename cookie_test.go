package fetch

import (
	"net/http"
	"testing"
)

func TestAddCookie(t *testing.T) {
	tests := []struct {
		name           string
		cookies        []*http.Cookie
		expectedCount  int
		expectedNames  []string
		expectedValues []string
	}{
		{
			name: "add single cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			expectedCount:  1,
			expectedNames:  []string{"session"},
			expectedValues: []string{"abc123"},
		},
		{
			name: "add multiple cookies",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "token", Value: "xyz789"},
			},
			expectedCount:  2,
			expectedNames:  []string{"session", "token"},
			expectedValues: []string{"abc123", "xyz789"},
		},
		{
			name: "cookie with path and domain",
			cookies: []*http.Cookie{
				{
					Name:   "secure",
					Value:  "value",
					Path:   "/api",
					Domain: ".example.com",
				},
			},
			expectedCount:  1,
			expectedNames:  []string{"secure"},
			expectedValues: []string{"value"},
		},
		{
			name:          "no cookies",
			cookies:       []*http.Cookie{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := AddCookie(tt.cookies...)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				cookies := req.Cookies()
				if len(cookies) != tt.expectedCount {
					t.Errorf("expected %d cookies, got %d", tt.expectedCount, len(cookies))
				}

				for i, expectedName := range tt.expectedNames {
					found := false
					for _, cookie := range cookies {
						if cookie.Name == expectedName {
							found = true
							if cookie.Value != tt.expectedValues[i] {
								t.Errorf("expected cookie %q to have value %q, got %q",
									expectedName, tt.expectedValues[i], cookie.Value)
							}
							break
						}
					}
					if !found {
						t.Errorf("expected cookie %q not found", expectedName)
					}
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestAddCookie_Duplicate(t *testing.T) {
	// Test that AddCookie preserves duplicate cookie names
	req, err := http.NewRequest("GET", "http://example.com/path", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// First add a cookie
	middleware1 := AddCookie(&http.Cookie{Name: "key", Value: "value1"})
	handler1 := middleware1(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return nil, nil
	}))
	handler1.Handle(&http.Client{}, req)

	// Then add another cookie with the same name
	middleware2 := AddCookie(&http.Cookie{Name: "key", Value: "value2"})
	handler2 := middleware2(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		cookies := req.Cookies()
		if len(cookies) != 2 {
			t.Errorf("expected 2 cookies, got %d", len(cookies))
		}

		values := make(map[string]bool)
		for _, c := range cookies {
			if c.Name == "key" {
				values[c.Value] = true
			}
		}

		if !values["value1"] || !values["value2"] {
			t.Errorf("expected both cookie values to be present, got %v", values)
		}
		return nil, nil
	}))

	handler2.Handle(&http.Client{}, req)
}

func TestDelAllCookies(t *testing.T) {
	tests := []struct {
		name          string
		setupCookies  []*http.Cookie
		expectedCount int
	}{
		{
			name: "delete all cookies",
			setupCookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "token", Value: "xyz789"},
			},
			expectedCount: 0,
		},
		{
			name:          "no cookies to delete",
			setupCookies:  []*http.Cookie{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			// Add cookies first
			for _, cookie := range tt.setupCookies {
				req.AddCookie(cookie)
			}

			middleware := DelAllCookies()
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Check that Cookie header is removed
				if cookieHeader := req.Header.Get("Cookie"); cookieHeader != "" {
					t.Errorf("expected Cookie header to be empty, got %q", cookieHeader)
				}
				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}
