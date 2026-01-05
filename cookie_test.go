package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookiesAdd(t *testing.T) {
	tests := []struct {
		name            string
		cookies         []*http.Cookie
		expectedCookies map[string]string
	}{
		{
			name: "single cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			expectedCookies: map[string]string{
				"session": "abc123",
			},
		},
		{
			name: "multiple cookies",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "auth", Value: "token456"},
				{Name: "user", Value: "john"},
			},
			expectedCookies: map[string]string{
				"session": "abc123",
				"auth":    "token456",
				"user":    "john",
			},
		},
		{
			name: "cookie with path and domain",
			cookies: []*http.Cookie{
				{
					Name:   "tracking",
					Value:  "xyz789",
					Path:   "/api",
					Domain: "example.com",
				},
			},
			expectedCookies: map[string]string{
				"tracking": "xyz789",
			},
		},
		{
			name: "cookie with special characters in value",
			cookies: []*http.Cookie{
				{Name: "data", Value: "value with spaces"},
			},
			expectedCookies: map[string]string{
				"data": "value with spaces",
			},
		},
		{
			name:            "empty cookies slice",
			cookies:         []*http.Cookie{},
			expectedCookies: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for name, expectedValue := range tt.expectedCookies {
					cookie, err := r.Cookie(name)
					require.NoError(t, err, "cookie %s should be present", name)
					assert.Equal(t, expectedValue, cookie.Value, "cookie %s value mismatch", name)
				}
				// Verify no unexpected cookies are present
				actualCookies := r.Cookies()
				assert.Len(t, actualCookies, len(tt.expectedCookies), "unexpected number of cookies")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Use(CookiesAdd(tt.cookies...))

			resp := req.Send("GET", server.URL)
			defer resp.Close()

			require.NoError(t, resp.Error)
			assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
		})
	}
}

func TestCookiesAdd_MultipleCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		assert.Len(t, cookies, 2)

		cookie1, err := r.Cookie("first")
		require.NoError(t, err)
		assert.Equal(t, "value1", cookie1.Value)

		cookie2, err := r.Cookie("second")
		require.NoError(t, err)
		assert.Equal(t, "value2", cookie2.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewDispatcher(nil)
	req := dispatcher.NewRequest().
		Use(CookiesAdd(&http.Cookie{Name: "first", Value: "value1"})).
		Use(CookiesAdd(&http.Cookie{Name: "second", Value: "value2"}))

	resp := req.Send("GET", server.URL)
	defer resp.Close()

	require.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
}

func TestCookiesRemove(t *testing.T) {
	tests := []struct {
		name             string
		initialCookies   []*http.Cookie
		expectAnyCookies bool
	}{
		{
			name: "remove existing cookies",
			initialCookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "auth", Value: "token456"},
			},
			expectAnyCookies: false,
		},
		{
			name:             "no initial cookies",
			initialCookies:   []*http.Cookie{},
			expectAnyCookies: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookies := r.Cookies()
				if tt.expectAnyCookies {
					assert.NotEmpty(t, cookies, "expected cookies to be present")
				} else {
					assert.Empty(t, cookies, "expected no cookies")
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().
				Use(CookiesAdd(tt.initialCookies...)).
				Use(CookiesRemove())

			resp := req.Send("GET", server.URL)
			defer resp.Close()

			require.NoError(t, resp.Error)
			assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
		})
	}
}

func TestCookiesRemove_WithSubsequentAdd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		assert.Len(t, cookies, 1, "should only have the cookie added after removal")

		cookie, err := r.Cookie("new")
		require.NoError(t, err)
		assert.Equal(t, "value", cookie.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewDispatcher(nil)
	req := dispatcher.NewRequest().
		Use(CookiesAdd(&http.Cookie{Name: "old", Value: "oldvalue"})).
		Use(CookiesRemove()).
		Use(CookiesAdd(&http.Cookie{Name: "new", Value: "value"}))

	resp := req.Send("GET", server.URL)
	defer resp.Close()

	require.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
}

func TestCookiesAdd_Integration(t *testing.T) {
	t.Run("global dispatcher cookies", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("global")
			require.NoError(t, err)
			assert.Equal(t, "globalvalue", cookie.Value)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dispatcher := NewDispatcher(nil)
		dispatcher.Use(CookiesAdd(&http.Cookie{Name: "global", Value: "globalvalue"}))

		req := dispatcher.NewRequest()
		resp := req.Send("GET", server.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})

	t.Run("request overrides global cookies", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookies := r.Cookies()
			assert.Len(t, cookies, 2)

			globalCookie, err := r.Cookie("global")
			require.NoError(t, err)
			assert.Equal(t, "globalvalue", globalCookie.Value)

			requestCookie, err := r.Cookie("request")
			require.NoError(t, err)
			assert.Equal(t, "requestvalue", requestCookie.Value)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dispatcher := NewDispatcher(nil)
		dispatcher.Use(CookiesAdd(&http.Cookie{Name: "global", Value: "globalvalue"}))

		req := dispatcher.NewRequest()
		req.Use(CookiesAdd(&http.Cookie{Name: "request", Value: "requestvalue"}))

		resp := req.Send("GET", server.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})
}
