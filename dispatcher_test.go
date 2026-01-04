package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDispatcher(t *testing.T) {
	tests := []struct {
		name                string
		client              *http.Client
		middlewares         []Middleware
		expectDefaultClient bool
	}{
		{
			name:                "nil client creates default",
			client:              nil,
			middlewares:         nil,
			expectDefaultClient: true,
		},
		{
			name: "custom client",
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
			middlewares:         nil,
			expectDefaultClient: false,
		},
		{
			name:   "with middlewares",
			client: nil,
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-Test", "test")
						return next.Handle(client, req)
					})
				},
			},
			expectDefaultClient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(tt.client, tt.middlewares...)

			assert.NotNil(t, dispatcher)
			assert.NotNil(t, dispatcher.Client())

			if tt.expectDefaultClient {
				assert.Equal(t, 30*time.Second, dispatcher.Client().Timeout)
			}

			if len(tt.middlewares) > 0 {
				assert.Equal(t, len(tt.middlewares), len(dispatcher.Middlewares()))
			}
		})
	}
}

func TestDispatcher_Client(t *testing.T) {
	tests := []struct {
		name   string
		client *http.Client
	}{
		{
			name:   "get default client",
			client: nil,
		},
		{
			name: "get custom client",
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(tt.client)
			client := dispatcher.Client()

			assert.NotNil(t, client)
		})
	}
}

func TestDispatcher_SetClient(t *testing.T) {
	tests := []struct {
		name      string
		newClient *http.Client
		shouldSet bool
	}{
		{
			name: "set valid client",
			newClient: &http.Client{
				Timeout: 15 * time.Second,
			},
			shouldSet: true,
		},
		{
			name:      "set nil client does nothing",
			newClient: nil,
			shouldSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			originalClient := dispatcher.Client()

			dispatcher.SetClient(tt.newClient)

			if tt.shouldSet {
				assert.NotEqual(t, originalClient, dispatcher.Client())
				assert.Equal(t, tt.newClient.Timeout, dispatcher.Client().Timeout)
			} else {
				assert.NotNil(t, dispatcher.Client())
			}
		})
	}
}

func TestDispatcher_Use(t *testing.T) {
	tests := []struct {
		name          string
		initialMws    []Middleware
		additionalMws []Middleware
		expectedCount int
	}{
		{
			name:       "add single middleware",
			initialMws: nil,
			additionalMws: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
			},
			expectedCount: 1,
		},
		{
			name: "add multiple middlewares",
			initialMws: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
			},
			additionalMws: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil, tt.initialMws...)
			dispatcher.Use(tt.additionalMws...)

			assert.Equal(t, tt.expectedCount, len(dispatcher.Middlewares()))
		})
	}
}

func TestDispatcher_Clone(t *testing.T) {
	tests := []struct {
		name            string
		setupDispatcher func() *Dispatcher
	}{
		{
			name: "clone with middlewares",
			setupDispatcher: func() *Dispatcher {
				d := NewDispatcher(nil)
				d.Use(func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-Original", "true")
						return next.Handle(client, req)
					})
				})
				return d
			},
		},
		{
			name: "clone empty dispatcher",
			setupDispatcher: func() *Dispatcher {
				return NewDispatcher(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setupDispatcher()
			cloned := original.Clone()

			assert.NotNil(t, cloned)
			assert.NotSame(t, original, cloned)
			assert.NotSame(t, original.Client(), cloned.Client())
			assert.Equal(t, len(original.Middlewares()), len(cloned.Middlewares()))
		})
	}
}

func TestDispatcher_Do(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		middlewares    []Middleware
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful request without middlewares",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			middlewares:    nil,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "successful request with middleware",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "test-value", r.Header.Get("X-Custom"))
					w.WriteHeader(http.StatusOK)
				}))
			},
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-Custom", "test-value")
						return next.Handle(client, req)
					})
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "multiple middlewares chain",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "first", r.Header.Get("X-First"))
					assert.Equal(t, "second", r.Header.Get("X-Second"))
					w.WriteHeader(http.StatusOK)
				}))
			},
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-First", "first")
						return next.Handle(client, req)
					})
				},
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-Second", "second")
						return next.Handle(client, req)
					})
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			dispatcher := NewDispatcher(nil)
			req, err := http.NewRequest("GET", server.URL, nil)
			require.NoError(t, err)

			resp, err := dispatcher.Do(req, tt.middlewares...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestDispatcher_NewRequest(t *testing.T) {
	tests := []struct {
		name       string
		dispatcher *Dispatcher
	}{
		{
			name:       "create new request",
			dispatcher: NewDispatcher(nil),
		},
		{
			name: "create request with middlewares",
			dispatcher: NewDispatcher(nil, func(next Handler) Handler {
				return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
					return next.Handle(client, req)
				})
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.dispatcher.NewRequest()

			assert.NotNil(t, request)
			assert.Equal(t, tt.dispatcher, request.dispatcher)
		})
	}
}

func TestDispatcher_Middlewares(t *testing.T) {
	tests := []struct {
		name          string
		middlewares   []Middleware
		expectedCount int
	}{
		{
			name:          "no middlewares",
			middlewares:   nil,
			expectedCount: 0,
		},
		{
			name: "single middleware",
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
			},
			expectedCount: 1,
		},
		{
			name: "multiple middlewares",
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						return next.Handle(client, req)
					})
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil, tt.middlewares...)
			middlewares := dispatcher.Middlewares()

			assert.Equal(t, tt.expectedCount, len(middlewares))
		})
	}
}
