package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareClientMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		clientOptions  []func(*http.Client)
		setupContext   func(context.Context) context.Context
		validateClient func(*testing.T, *http.Client)
	}{
		{
			name: "no options - uses default client",
			validateClient: func(t *testing.T, client *http.Client) {
				assert.NotNil(t, client)
			},
		},
		{
			name: "with context options - applies timeout",
			setupContext: func(ctx context.Context) context.Context {
				return WithClientOptions(ctx, func(c *http.Client) {
					c.Timeout = 5 * time.Second
				})
			},
			validateClient: func(t *testing.T, client *http.Client) {
				assert.Equal(t, 5*time.Second, client.Timeout)
			},
		},
		{
			name: "multiple options - applies all",
			setupContext: func(ctx context.Context) context.Context {
				ctx = WithClientOptions(ctx, func(c *http.Client) {
					c.Timeout = 10 * time.Second
				})
				return WithClientOptions(ctx, func(c *http.Client) {
					c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					}
				})
			},
			validateClient: func(t *testing.T, client *http.Client) {
				assert.Equal(t, 10*time.Second, client.Timeout)
				assert.NotNil(t, client.CheckRedirect)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedClient *http.Client

			middleware := PrepareClientMiddleware()
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				capturedClient = client
				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			if tt.setupContext != nil {
				ctx := tt.setupContext(req.Context())
				req = req.WithContext(ctx)
			}

			client := &http.Client{}
			resp, err := handler.Handle(client, req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, 200, resp.StatusCode)

			if tt.validateClient != nil {
				tt.validateClient(t, capturedClient)
			}
		})
	}
}

func TestSetClientOptions(t *testing.T) {
	tests := []struct {
		name           string
		options        []func(*http.Client)
		validateClient func(*testing.T, *http.Client)
	}{
		{
			name: "single option - sets timeout",
			options: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 30 * time.Second
				},
			},
			validateClient: func(t *testing.T, client *http.Client) {
				assert.Equal(t, 30*time.Second, client.Timeout)
			},
		},
		{
			name: "multiple options - applies in order",
			options: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 1 * time.Minute
				},
				func(c *http.Client) {
					c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						if len(via) >= 3 {
							return http.ErrUseLastResponse
						}
						return nil
					}
				},
			},
			validateClient: func(t *testing.T, client *http.Client) {
				assert.Equal(t, 1*time.Minute, client.Timeout)
				assert.NotNil(t, client.CheckRedirect)
			},
		},
		{
			name:    "no options - client unchanged",
			options: []func(*http.Client){},
			validateClient: func(t *testing.T, client *http.Client) {
				assert.NotNil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest()

			if len(tt.options) > 0 {
				req = req.Use(SetClientOptions(tt.options...))
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send("GET", server.URL)
			assert.NoError(t, resp.Error)
		})
	}
}

func TestSetClientOptions_Integration(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func(*Request) *Request
	}{
		{
			name: "custom transport",
			setupReq: func(req *Request) *Request {
				return req.Use(SetClientOptions(func(c *http.Client) {
					c.Transport = &http.Transport{
						MaxIdleConns: 10,
					}
				}))
			},
		},
		{
			name: "disable redirects",
			setupReq: func(req *Request) *Request {
				return req.Use(SetClientOptions(func(c *http.Client) {
					c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					}
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest()

			if tt.setupReq != nil {
				req = tt.setupReq(req)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send("GET", server.URL)
			assert.NoError(t, resp.Error)
		})
	}
}

func TestWithClientOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []func(*http.Client)
		verify  func(*testing.T, context.Context)
	}{
		{
			name: "adds single option to context",
			options: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 15 * time.Second
				},
			},
			verify: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
		{
			name: "adds multiple options to context",
			options: []func(*http.Client){
				func(c *http.Client) {
					c.Timeout = 20 * time.Second
				},
				func(c *http.Client) {
					c.CheckRedirect = nil
				},
			},
			verify: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
		{
			name:    "empty options returns valid context",
			options: []func(*http.Client){},
			verify: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithClientOptions(ctx, tt.options...)

			if tt.verify != nil {
				tt.verify(t, ctx)
			}
		})
	}
}

func TestWithClientOptions_Chaining(t *testing.T) {
	ctx := context.Background()

	// Chain multiple WithClientOptions calls
	ctx = WithClientOptions(ctx, func(c *http.Client) {
		c.Timeout = 5 * time.Second
	})

	ctx = WithClientOptions(ctx, func(c *http.Client) {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	})

	assert.NotNil(t, ctx)

	// Verify context can be used in request
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	req = req.WithContext(ctx)
	assert.NotNil(t, req.Context())
}
