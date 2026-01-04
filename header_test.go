package fetch

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareHeaderMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		options         []func(*HeaderOptions)
		expectedHeaders map[string]string
	}{
		{
			name:            "no headers configured",
			options:         []func(*HeaderOptions){},
			expectedHeaders: map[string]string{},
		},
		{
			name: "single header",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Custom", "value")
				},
			},
			expectedHeaders: map[string]string{
				"X-Custom": "value",
			},
		},
		{
			name: "multiple headers",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Custom", "value1")
					opts.Header.Set("X-Another", "value2")
				},
			},
			expectedHeaders: map[string]string{
				"X-Custom":  "value1",
				"X-Another": "value2",
			},
		},
		{
			name: "headers from multiple option functions",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-First", "first")
				},
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Second", "second")
				},
			},
			expectedHeaders: map[string]string{
				"X-First":  "first",
				"X-Second": "second",
			},
		},
		{
			name: "common request headers",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("User-Agent", "MyApp/1.0")
					opts.Header.Set("Accept", "application/json")
					opts.Header.Set("Content-Type", "application/json")
				},
			},
			expectedHeaders: map[string]string{
				"User-Agent":   "MyApp/1.0",
				"Accept":       "application/json",
				"Content-Type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := PrepareHeaderMiddleware()
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				for name, expectedValue := range tt.expectedHeaders {
					actualValue := req.Header.Get(name)
					assert.Equal(t, expectedValue, actualValue, "header %s mismatch", name)
				}
				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			// Apply options to context
			if len(tt.options) > 0 {
				ctx := req.Context()
				for _, opt := range tt.options {
					ctx = WithHeaderOptions(ctx, opt)
				}
				req = req.WithContext(ctx)
			}

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestSetHeaderOptions(t *testing.T) {
	tests := []struct {
		name            string
		options         []func(*HeaderOptions)
		expectedHeaders map[string]string
	}{
		{
			name: "single option function",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Test", "value")
				},
			},
			expectedHeaders: map[string]string{
				"X-Test": "value",
			},
		},
		{
			name: "multiple option functions",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-First", "val1")
				},
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Second", "val2")
				},
			},
			expectedHeaders: map[string]string{
				"X-First":  "val1",
				"X-Second": "val2",
			},
		},
		{
			name: "authentication headers",
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("Authorization", "Bearer token123")
					opts.Header.Set("X-API-Key", "api-key-456")
				},
			},
			expectedHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"X-API-Key":     "api-key-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHeaderMW := SetHeaderOptions(tt.options...)
			prepareMW := PrepareHeaderMiddleware()

			// Compose middlewares: SetHeaderOptions -> PrepareHeaderMiddleware -> Handler
			handler := setHeaderMW(prepareMW(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				for name, expectedValue := range tt.expectedHeaders {
					actualValue := req.Header.Get(name)
					assert.Equal(t, expectedValue, actualValue, "header %s mismatch", name)
				}
				return &http.Response{StatusCode: 200}, nil
			})))

			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestWithHeaderOptions(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		options  []func(*HeaderOptions)
		validate func(t *testing.T, ctx context.Context)
	}{
		{
			name: "add options to context",
			ctx:  context.Background(),
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Context", "value")
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
				// Verify context contains header options
				val, ok := prepareHeaderKey.GetValue(ctx)
				assert.True(t, ok)
				assert.Len(t, val, 1)
			},
		},
		{
			name: "nil context creates background",
			ctx:  nil,
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-Header", "val")
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
		{
			name: "multiple options accumulated",
			ctx:  context.Background(),
			options: []func(*HeaderOptions){
				func(opts *HeaderOptions) {
					opts.Header.Set("X-H1", "v1")
				},
				func(opts *HeaderOptions) {
					opts.Header.Set("X-H2", "v2")
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				val, ok := prepareHeaderKey.GetValue(ctx)
				assert.True(t, ok)
				assert.Len(t, val, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithHeaderOptions(tt.ctx, tt.options...)
			tt.validate(t, ctx)
		})
	}
}

func TestHeaderOptions_Integration(t *testing.T) {
	t.Run("headers from context", func(t *testing.T) {
		middleware := PrepareHeaderMiddleware()
		handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, "ctx_value", req.Header.Get("X-Context-Header"))
			return &http.Response{StatusCode: 200}, nil
		}))

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := WithHeaderOptions(req.Context(), func(opts *HeaderOptions) {
			opts.Header.Set("X-Context-Header", "ctx_value")
		})
		req = req.WithContext(ctx)

		client := &http.Client{}
		_, err := handler.Handle(client, req)
		assert.NoError(t, err)
	})

	t.Run("headers from SetHeaderOptions middleware", func(t *testing.T) {
		setHeaderMW := SetHeaderOptions(func(opts *HeaderOptions) {
			opts.Header.Set("X-Middleware-Header", "mw_value")
		})
		prepareMW := PrepareHeaderMiddleware()

		handler := setHeaderMW(prepareMW(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, "mw_value", req.Header.Get("X-Middleware-Header"))
			return &http.Response{StatusCode: 200}, nil
		})))

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		client := &http.Client{}
		_, err := handler.Handle(client, req)
		assert.NoError(t, err)
	})

	t.Run("overwrite existing header", func(t *testing.T) {
		middleware := PrepareHeaderMiddleware()
		handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, "new_value", req.Header.Get("X-Key"))
			return &http.Response{StatusCode: 200}, nil
		}))

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Key", "old_value")

		ctx := WithHeaderOptions(req.Context(), func(opts *HeaderOptions) {
			opts.Header.Set("X-Key", "new_value")
		})
		req = req.WithContext(ctx)

		client := &http.Client{}
		_, err := handler.Handle(client, req)
		assert.NoError(t, err)
	})
}
