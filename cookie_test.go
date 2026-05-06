package fetch

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareCookieMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		options         []func(*CookieOptions)
		expectedCookies []*http.Cookie
	}{
		{
			name:            "no cookies",
			options:         []func(*CookieOptions){},
			expectedCookies: []*http.Cookie{},
		},
		{
			name: "single cookie",
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "session", Value: "token123"})
				},
			},
			expectedCookies: []*http.Cookie{
				{Name: "session", Value: "token123"},
			},
		},
		{
			name: "multiple cookies",
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "session", Value: "token123"})
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "user", Value: "john"})
				},
			},
			expectedCookies: []*http.Cookie{
				{Name: "session", Value: "token123"},
				{Name: "user", Value: "john"},
			},
		},
		{
			name: "cookies from multiple option functions",
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "cookie1", Value: "value1"})
				},
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "cookie2", Value: "value2"})
				},
			},
			expectedCookies: []*http.Cookie{
				{Name: "cookie1", Value: "value1"},
				{Name: "cookie2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := PrepareCookieMiddleware()
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				cookies := req.Cookies()
				assert.Len(t, cookies, len(tt.expectedCookies))

				for i, expectedCookie := range tt.expectedCookies {
					if i < len(cookies) {
						assert.Equal(t, expectedCookie.Name, cookies[i].Name)
						assert.Equal(t, expectedCookie.Value, cookies[i].Value)
					}
				}

				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			// Apply options to context
			if len(tt.options) > 0 {
				ctx := req.Context()
				for _, opt := range tt.options {
					ctx = WithCookieOptions(ctx, opt)
				}
				req = req.WithContext(ctx)
			}

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestSetCookieOptions(t *testing.T) {
	tests := []struct {
		name            string
		options         []func(*CookieOptions)
		expectedCookies []*http.Cookie
	}{
		{
			name: "single option function",
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "test", Value: "value"})
				},
			},
			expectedCookies: []*http.Cookie{
				{Name: "test", Value: "value"},
			},
		},
		{
			name: "multiple option functions",
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "first", Value: "val1"})
				},
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "second", Value: "val2"})
				},
			},
			expectedCookies: []*http.Cookie{
				{Name: "first", Value: "val1"},
				{Name: "second", Value: "val2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCookieMW := SetCookieOptions(tt.options...)
			prepareMW := PrepareCookieMiddleware()

			// Compose middlewares: SetCookieOptions -> PrepareCookieMiddleware -> Handler
			handler := setCookieMW(prepareMW(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				cookies := req.Cookies()
				assert.Len(t, cookies, len(tt.expectedCookies))

				for i, expectedCookie := range tt.expectedCookies {
					if i < len(cookies) {
						assert.Equal(t, expectedCookie.Name, cookies[i].Name)
						assert.Equal(t, expectedCookie.Value, cookies[i].Value)
					}
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

func TestWithCookieOptions(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		options  []func(*CookieOptions)
		validate func(t *testing.T, ctx context.Context)
	}{
		{
			name: "add options to context",
			ctx:  context.Background(),
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "test", Value: "value"})
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
				// Verify context contains cookie options
				val, ok := prepareCookieKey.GetValue(ctx)
				assert.True(t, ok)
				assert.Len(t, val, 1)
			},
		},
		{
			name: "nil context creates background",
			ctx:  nil,
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "cookie", Value: "val"})
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				assert.NotNil(t, ctx)
			},
		},
		{
			name: "multiple options accumulated",
			ctx:  context.Background(),
			options: []func(*CookieOptions){
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "c1", Value: "v1"})
				},
				func(opts *CookieOptions) {
					opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "c2", Value: "v2"})
				},
			},
			validate: func(t *testing.T, ctx context.Context) {
				val, ok := prepareCookieKey.GetValue(ctx)
				assert.True(t, ok)
				assert.Len(t, val, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithCookieOptions(tt.ctx, tt.options...)
			tt.validate(t, ctx)
		})
	}
}

func TestCookieOptions_Integration(t *testing.T) {
	t.Run("cookies from context", func(t *testing.T) {
		middleware := PrepareCookieMiddleware()
		handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			cookies := req.Cookies()
			require.Len(t, cookies, 1)
			assert.Equal(t, "ctx_cookie", cookies[0].Name)
			assert.Equal(t, "ctx_value", cookies[0].Value)
			return &http.Response{StatusCode: 200}, nil
		}))

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := WithCookieOptions(req.Context(), func(opts *CookieOptions) {
			opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "ctx_cookie", Value: "ctx_value"})
		})
		req = req.WithContext(ctx)

		client := &http.Client{}
		_, err := handler.Handle(client, req)
		assert.NoError(t, err)
	})

	t.Run("cookies from SetCookieOptions middleware", func(t *testing.T) {
		setCookieMW := SetCookieOptions(func(opts *CookieOptions) {
			opts.Cookies = append(opts.Cookies, &http.Cookie{Name: "mw_cookie", Value: "mw_value"})
		})
		prepareMW := PrepareCookieMiddleware()

		handler := setCookieMW(prepareMW(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			cookies := req.Cookies()
			require.Len(t, cookies, 1)
			assert.Equal(t, "mw_cookie", cookies[0].Name)
			assert.Equal(t, "mw_value", cookies[0].Value)
			return &http.Response{StatusCode: 200}, nil
		})))

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		client := &http.Client{}
		_, err := handler.Handle(client, req)
		assert.NoError(t, err)
	})
}
