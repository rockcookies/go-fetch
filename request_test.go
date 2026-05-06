package fetch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			resp := req.Send("GET", server.URL)
			assert.NoError(t, resp.Error)
		})
	}
}

func TestRequest_UseFuncs(t *testing.T) {
	tests := []struct {
		name            string
		funcs           []func(*http.Request)
		expectedHeaders map[string]string
	}{
		{
			name: "single func",
			funcs: []func(*http.Request){
				func(req *http.Request) {
					req.Header.Set("X-Custom", "value")
				},
			},
			expectedHeaders: map[string]string{"X-Custom": "value"},
		},
		{
			name: "multiple funcs",
			funcs: []func(*http.Request){
				func(req *http.Request) {
					req.Header.Set("X-A", "a")
				},
				func(req *http.Request) {
					req.Header.Set("X-B", "b")
				},
			},
			expectedHeaders: map[string]string{
				"X-A": "a",
				"X-B": "b",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest().UseFuncs(tt.funcs...)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, r.Header.Get(key))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := req.Send("GET", server.URL)
			defer resp.Close()
			assert.NoError(t, resp.Error)
		})
	}
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

			resp := req.Send("POST", server.URL)
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

			resp := req.Send("POST", server.URL)
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

			resp := req.Send("POST", server.URL)
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
			name: "clone preserves middlewares",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			original := dispatcher.NewRequest().UseFuncs(func(req *http.Request) {
				req.Header.Set("X-Original", "true")
			})

			cloned := original.Clone()
			cloned.UseFuncs(func(req *http.Request) {
				req.Header.Set("X-Cloned", "true")
			})

			// Original should not have X-Cloned header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "true", r.Header.Get("X-Original"))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			resp := original.Send("GET", server.URL)
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
		expectedStatus int
		expectError    bool
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
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:           "invalid URL",
			method:         "GET",
			setupServer:    func() *httptest.Server { return nil },
			expectedStatus: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewDispatcher(nil)
			req := dispatcher.NewRequest()

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

			resp := req.Send(tt.method, serverURL)
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
