package fetch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFuncs(t *testing.T) {
	tests := []struct {
		name          string
		clientFuncs   []func(*http.Client)
		verifyClient  func(*testing.T, *http.Client)
		verifyRequest func(*testing.T, *http.Request)
		shouldSucceed bool
	}{
		{
			name: "set timeout",
			clientFuncs: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 5 * time.Second
				},
			},
			verifyClient: func(t *testing.T, c *http.Client) {
				assert.Equal(t, 5*time.Second, c.Timeout)
			},
			shouldSucceed: true,
		},
		{
			name: "disable redirects",
			clientFuncs: []func(*http.Client){
				func(c *http.Client) {
					c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					}
				},
			},
			verifyClient: func(t *testing.T, c *http.Client) {
				assert.NotNil(t, c.CheckRedirect)
			},
			shouldSucceed: true,
		},
		{
			name: "multiple client modifications",
			clientFuncs: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 10 * time.Second
				},
				func(c *http.Client) {
					c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						if len(via) >= 2 {
							return http.ErrUseLastResponse
						}
						return nil
					}
				},
			},
			verifyClient: func(t *testing.T, c *http.Client) {
				assert.Equal(t, 10*time.Second, c.Timeout)
				assert.NotNil(t, c.CheckRedirect)
			},
			shouldSucceed: true,
		},
		{
			name:          "no client functions",
			clientFuncs:   []func(*http.Client){},
			shouldSucceed: true,
		},
		{
			name:          "nil client functions slice",
			clientFuncs:   nil,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.verifyRequest != nil {
					tt.verifyRequest(t, r)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Use(ClientFuncs(tt.clientFuncs...))

			resp := req.Send("GET", server.URL)
			defer resp.Close()

			if tt.shouldSucceed {
				require.NoError(t, resp.Error)
				assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
			} else {
				require.Error(t, resp.Error)
			}

			// Verify client modifications if verifyClient is provided
			if tt.verifyClient != nil && tt.shouldSucceed {
				// Create a new client and apply the functions to verify they work
				testClient := &http.Client{}
				for _, f := range tt.clientFuncs {
					f(testClient)
				}
				tt.verifyClient(t, testClient)
			}
		})
	}
}

func TestClientFuncs_Integration(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" && redirectCount < 1 {
			redirectCount++
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final"))
	}))
	defer server.Close()

	t.Run("prevent redirect", func(t *testing.T) {
		redirectCount = 0
		dispatcher := NewDispatcher(nil)
		req := dispatcher.NewRequest().Use(ClientFuncs(func(c *http.Client) {
			c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}))

		resp := req.Send("GET", server.URL+"/redirect")
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusFound, resp.RawResponse.StatusCode)
		assert.Equal(t, 1, redirectCount)
	})

	t.Run("allow redirect", func(t *testing.T) {
		redirectCount = 0
		dispatcher := NewDispatcher(nil)
		req := dispatcher.NewRequest() // No ClientFuncs - default behavior follows redirects

		resp := req.Send("GET", server.URL+"/redirect")
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
		assert.Equal(t, "final", resp.String())
	})
}

func TestClientFuncs_WithTimeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	t.Run("timeout triggers", func(t *testing.T) {
		dispatcher := NewDispatcher(nil)
		req := dispatcher.NewRequest().Use(ClientFuncs(func(c *http.Client) {
			c.Timeout = 50 * time.Millisecond
		}))

		resp := req.Send("GET", slowServer.URL)
		defer resp.Close()

		require.Error(t, resp.Error)
		// Error message contains "deadline" or "timeout"
		errMsg := resp.Error.Error()
		assert.True(t,
			strings.Contains(errMsg, "deadline") || strings.Contains(errMsg, "timeout"),
			"expected timeout or deadline error, got: %s", errMsg)
	})

	t.Run("sufficient timeout succeeds", func(t *testing.T) {
		dispatcher := NewDispatcher(nil)
		req := dispatcher.NewRequest().Use(ClientFuncs(func(c *http.Client) {
			c.Timeout = 500 * time.Millisecond
		}))

		resp := req.Send("GET", slowServer.URL)
		defer resp.Close()

		require.NoError(t, resp.Error)
		assert.Equal(t, http.StatusOK, resp.RawResponse.StatusCode)
	})
}
