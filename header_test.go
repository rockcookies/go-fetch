package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderFuncs(t *testing.T) {
	tests := []struct {
		name            string
		headerFuncs     []func(http.Header)
		expectedHeaders map[string][]string
	}{
		{
			name: "set single header",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Set("X-Custom", "value")
				},
			},
			expectedHeaders: map[string][]string{
				"X-Custom": {"value"},
			},
		},
		{
			name: "set multiple headers",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Set("User-Agent", "MyApp/1.0")
					h.Set("Accept", "application/json")
					h.Set("Authorization", "Bearer token123")
				},
			},
			expectedHeaders: map[string][]string{
				"User-Agent":    {"MyApp/1.0"},
				"Accept":        {"application/json"},
				"Authorization": {"Bearer token123"},
			},
		},
		{
			name: "add multiple values for same header",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Add("X-Custom", "value1")
					h.Add("X-Custom", "value2")
					h.Add("X-Custom", "value3")
				},
			},
			expectedHeaders: map[string][]string{
				"X-Custom": {"value1", "value2", "value3"},
			},
		},
		{
			name: "delete header",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Set("X-Temp", "temporary")
					h.Del("X-Temp")
				},
			},
			expectedHeaders: map[string][]string{},
		},
		{
			name: "multiple functions in sequence",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Set("X-First", "first")
				},
				func(h http.Header) {
					h.Set("X-Second", "second")
				},
				func(h http.Header) {
					h.Set("X-Third", "third")
				},
			},
			expectedHeaders: map[string][]string{
				"X-First":  {"first"},
				"X-Second": {"second"},
				"X-Third":  {"third"},
			},
		},
		{
			name: "override existing header",
			headerFuncs: []func(http.Header){
				func(h http.Header) {
					h.Set("X-Value", "original")
					h.Set("X-Value", "updated")
				},
			},
			expectedHeaders: map[string][]string{
				"X-Value": {"updated"},
			},
		},
		{
			name:            "empty functions slice",
			headerFuncs:     []func(http.Header){},
			expectedHeaders: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for name, expectedValues := range tt.expectedHeaders {
					actualValues := r.Header[name]
					assert.Equal(t, expectedValues, actualValues, "header %s mismatch", name)
				}

				// Verify headers that should not be present
				for headerName := range r.Header {
					if _, expected := tt.expectedHeaders[headerName]; !expected {
						// Skip standard headers added by http client
						standardHeaders := map[string]bool{
							"Accept-Encoding": true,
							"User-Agent":      true,
							"Content-Length":  true,
							"Content-Type":    true,
							"Host":            true,
						}
						if !standardHeaders[headerName] {
							t.Errorf("unexpected header present: %s", headerName)
						}
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Use(HeaderFuncs(tt.headerFuncs...))

			resp := req.Send("GET", server.URL)
			defer resp.Close()

			require.NoError(t, resp.Error)
			assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
		})
	}
}

func TestHeaderFuncs_WithNilSlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewDispatcher(nil)
	req := dispatcher.NewRequest().Use(HeaderFuncs(nil...))

	resp := req.Send("GET", server.URL)
	defer resp.Close()

	require.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
}

func TestHeaderFuncs_MultipleMiddlewares(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value1", r.Header.Get("X-First"))
		assert.Equal(t, "value2", r.Header.Get("X-Second"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewDispatcher(nil)
	req := dispatcher.NewRequest().
		Use(HeaderFuncs(func(h http.Header) {
			h.Set("X-First", "value1")
		})).
		Use(HeaderFuncs(func(h http.Header) {
			h.Set("X-Second", "value2")
		}))

	resp := req.Send("GET", server.URL)
	defer resp.Close()

	require.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
}

func TestHeaderFuncs_Integration(t *testing.T) {
	t.Run("global dispatcher headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "MyApp/1.0", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dispatcher := NewDispatcher(nil)
		dispatcher.Use(HeaderFuncs(func(h http.Header) {
			h.Set("User-Agent", "MyApp/1.0")
			h.Set("Accept", "application/json")
		}))

		req := dispatcher.NewRequest()
		resp := req.Send("GET", server.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})

	t.Run("request headers override dispatcher headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "RequestApp/2.0", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dispatcher := NewDispatcher(nil)
		dispatcher.Use(HeaderFuncs(func(h http.Header) {
			h.Set("User-Agent", "DispatcherApp/1.0")
			h.Set("Accept", "application/json")
		}))

		req := dispatcher.NewRequest()
		req.Use(HeaderFuncs(func(h http.Header) {
			h.Set("User-Agent", "RequestApp/2.0")
		}))

		resp := req.Send("GET", server.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})

	t.Run("complex header manipulation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			assert.Equal(t, []string{"gzip", "deflate"}, r.Header["Accept-Encoding"])
			assert.Empty(t, r.Header.Get("X-Removed"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dispatcher := NewDispatcher(nil)
		req := dispatcher.NewRequest().Use(HeaderFuncs(
			func(h http.Header) {
				h.Set("Authorization", "Bearer token123")
				h.Set("X-Removed", "will-be-removed")
			},
			func(h http.Header) {
				h.Del("X-Removed")
				h.Del("Accept-Encoding") // Remove default
				h.Add("Accept-Encoding", "gzip")
				h.Add("Accept-Encoding", "deflate")
			},
		))

		resp := req.Send("GET", server.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})
}

func TestHeaderFuncs_CaseSensitivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HTTP headers are case-insensitive, but Go's http.Header canonicalizes them
		assert.Equal(t, "value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "value", r.Header.Get("x-custom-header"))
		assert.Equal(t, "value", r.Header.Get("X-CUSTOM-HEADER"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcher := NewDispatcher(nil)
	req := dispatcher.NewRequest().Use(HeaderFuncs(func(h http.Header) {
		h.Set("x-custom-header", "value")
	}))

	resp := req.Send("GET", server.URL)
	defer resp.Close()

	require.NoError(t, resp.Error)
	assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
}
