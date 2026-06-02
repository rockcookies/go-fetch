package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ctxKey struct{}

func TestRequestFunc(t *testing.T) {
	tests := []struct {
		name    string
		fn      RequestFunc
		setup   func(*http.Request) *http.Request
		check   func(t *testing.T, req *http.Request)
	}{
		{
			name: "nil func",
			fn:   nil,
			setup: func(req *http.Request) *http.Request {
				return req
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "GET", req.Method)
			},
		},
		{
			name: "in-place mutation with nil return",
			fn: func(req *http.Request) *http.Request {
				req.Header.Set("X-Custom", "value")
				return nil
			},
			setup: func(req *http.Request) *http.Request {
				return req
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "value", req.Header.Get("X-Custom"))
			},
		},
		{
			name: "replace request",
			fn: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), ctxKey{}, "trace")
				return req.WithContext(ctx)
			},
			setup: func(req *http.Request) *http.Request {
				return req
			},
			check: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "trace", req.Context().Value(ctxKey{}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
			require.NoError(t, err)
			req = tt.setup(req)
			result := tt.fn.Apply(req)
			tt.check(t, result)
		})
	}
}

func TestRequest_Use(t *testing.T) {
	tests := []struct {
		name            string
		middlewares     []Middleware
		expectedHeaders map[string]string
	}{
		{
			name: "single middleware",
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						req.Header.Set("X-Test", "value1")
						return next.Handle(client, req)
					})
				},
			},
			expectedHeaders: map[string]string{"X-Test": "value1"},
		},
		{
			name: "multiple middlewares",
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
			expectedHeaders: map[string]string{
				"X-First":  "first",
				"X-Second": "second",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Use(tt.middlewares...)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, r.Header.Get(key))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send(context.Background(), "GET", server.URL)
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_Pre(t *testing.T) {
	dispatcher := NewDispatcher(nil)

	preCalled := false
	useCalled := false

	req := dispatcher.NewRequest().
		Use(func(next Handler) Handler {
			return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				useCalled = true
				assert.True(t, preCalled, "Pre middleware should run before Use middleware")
				req.Header.Set("X-Use", "use")
				return next.Handle(client, req)
			})
		}).
		Pre(func(next Handler) Handler {
			return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				preCalled = true
				req.Header.Set("X-Pre", "pre")
				return next.Handle(client, req)
			})
		})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "pre", r.Header.Get("X-Pre"))
		assert.Equal(t, "use", r.Header.Get("X-Use"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp := req.Send(context.Background(), "GET", server.URL)
	defer resp.Close()
	assert.NoError(t, resp.Error)
	assert.True(t, preCalled)
	assert.True(t, useCalled)
}

func TestRequest_UseFuncs(t *testing.T) {
	tests := []struct {
		name            string
		funcs           []RequestFunc
		expectedHeaders map[string]string
		checkMiddleware bool
	}{
		{
			name: "single func",
			funcs: []RequestFunc{
				func(req *http.Request) *http.Request {
					req.Header.Set("X-Custom", "value")
					return nil
				},
			},
			expectedHeaders: map[string]string{"X-Custom": "value"},
		},
		{
			name: "multiple funcs",
			funcs: []RequestFunc{
				func(req *http.Request) *http.Request {
					req.Header.Set("X-A", "a")
					return nil
				},
				func(req *http.Request) *http.Request {
					req.Header.Set("X-B", "b")
					return nil
				},
			},
			expectedHeaders: map[string]string{
				"X-A": "a",
				"X-B": "b",
			},
		},
		{
			name: "replace request",
			funcs: []RequestFunc{
				func(req *http.Request) *http.Request {
					return req.WithContext(context.WithValue(req.Context(), ctxKey{}, "replaced"))
				},
			},
			expectedHeaders: map[string]string{},
			checkMiddleware: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().UseFuncs(tt.funcs...)
			if tt.checkMiddleware {
				req = req.Use(func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						assert.Equal(t, "replaced", req.Context().Value(ctxKey{}))
						return next.Handle(client, req)
					})
				})
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, r.Header.Get(key))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send(context.Background(), "GET", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_PreFuncs(t *testing.T) {
	dispatcher := NewDispatcher(nil)

	req := dispatcher.NewRequest().
		UseFuncs(func(req *http.Request) *http.Request {
			req.Header.Set("X-Use", "use")
			return nil
		}).
		PreFuncs(func(req *http.Request) *http.Request {
			req.Header.Set("X-Pre", "pre")
			return nil
		})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "pre", r.Header.Get("X-Pre"))
		assert.Equal(t, "use", r.Header.Get("X-Use"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp := req.Send(context.Background(), "GET", server.URL)
	defer resp.Close()
	assert.NoError(t, resp.Error)
}

func TestRequest_FuncsAndMiddlewareOrder(t *testing.T) {
	dispatcher := NewDispatcher(nil)

	funcsApplied := false
	req := dispatcher.NewRequest().
		UseFuncs(func(req *http.Request) *http.Request {
			req.Header.Set("X-Func", "func")
			return nil
		}).
		Use(func(next Handler) Handler {
			return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				assert.Equal(t, "func", req.Header.Get("X-Func"), "funcs should run before middleware")
				funcsApplied = true
				req.Header.Set("X-MW", "mw")
				return next.Handle(client, req)
			})
		})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "func", r.Header.Get("X-Func"))
		assert.Equal(t, "mw", r.Header.Get("X-MW"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp := req.Send(context.Background(), "GET", server.URL)
	defer resp.Close()
	assert.NoError(t, resp.Error)
	assert.True(t, funcsApplied)
}

func TestRequest_Body(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectedBody string
	}{
		{
			name:         "simple text body",
			body:         "test body",
			expectedBody: "test body",
		},
		{
			name:         "empty body",
			body:         "",
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Body(strings.NewReader(tt.body))

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := make([]byte, len(tt.expectedBody))
				r.Body.Read(body)
				assert.Equal(t, tt.expectedBody, string(body))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send(context.Background(), "POST", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_JSON(t *testing.T) {
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name                string
		data                any
		expectedContentType string
	}{
		{
			name:                "struct JSON",
			data:                TestData{Name: "test", Value: 123},
			expectedContentType: "application/json",
		},
		{
			name:                "string JSON",
			data:                `{"key":"value"}`,
			expectedContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().JSON(tt.data)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedContentType, r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send(context.Background(), "POST", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_Form(t *testing.T) {
	tests := []struct {
		name                string
		form                url.Values
		expectedContentType string
	}{
		{
			name: "simple form",
			form: url.Values{
				"username": []string{"john"},
				"password": []string{"secret"},
			},
			expectedContentType: "application/x-www-form-urlencoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().Form(tt.form)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedContentType, r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send(context.Background(), "POST", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_Clone(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "clone preserves funcs independently",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			original := dispatcher.NewRequest().PreFuncs(func(req *http.Request) *http.Request {
				req.Header.Set("X-Original", "true")
				return nil
			})

			cloned := original.Clone()
			cloned.UseFuncs(func(req *http.Request) *http.Request {
				req.Header.Set("X-Cloned", "true")
				return nil
			})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "true", r.Header.Get("X-Original"))
				assert.Empty(t, r.Header.Get("X-Cloned"))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := original.Send(context.Background(), "GET", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_Send(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupServer    func() *httptest.Server
		setupCtx       func() context.Context
		expectedStatus int
		expectError    bool
		checkCtx       bool
	}{
		{
			name:   "successful GET request",
			method: "GET",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "GET", r.Method)
					w.WriteHeader(http.StatusOK)
				}))
			},
			setupCtx:       func() context.Context { return context.Background() },
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "successful POST request",
			method: "POST",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "POST", r.Method)
					w.WriteHeader(http.StatusCreated)
				}))
			},
			setupCtx:       func() context.Context { return context.Background() },
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:   "context attached to request",
			method: "GET",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), ctxKey{}, "trace-id")
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			checkCtx:       true,
		},
		{
			name:           "invalid URL",
			method:         "GET",
			setupServer:    func() *httptest.Server { return nil },
			setupCtx:       func() context.Context { return context.Background() },
			expectedStatus: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest()
			if tt.checkCtx {
				req = req.Use(func(next Handler) Handler {
					return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
						assert.Equal(t, "trace-id", req.Context().Value(ctxKey{}))
						return next.Handle(client, req)
					})
				})
			}

			var serverURL string
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					serverURL = server.URL
				} else {
					serverURL = "://invalid-url"
				}
			} else {
				serverURL = "://invalid-url"
			}

			resp := req.Send(tt.setupCtx(), tt.method, serverURL)
			defer resp.Close()

			if tt.expectError {
				assert.Error(t, resp.Error)
			} else {
				assert.NoError(t, resp.Error)
				assert.Equal(t, tt.expectedStatus, resp.RawResponse.StatusCode)
			}
		})
	}
}

func TestRequest_Do(t *testing.T) {
	tests := []struct {
		name        string
		setupReq    func() *http.Request
		expectError bool
	}{
		{
			name: "successful request",
			setupReq: func() *http.Request {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				req, _ := http.NewRequest("GET", server.URL, nil)
				return req
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			request := dispatcher.NewRequest()

			httpReq := tt.setupReq()
			resp, err := request.Do(httpReq)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}
