package fetch

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerFunc(t *testing.T) {
	tests := []struct {
		name        string
		handlerFunc HandlerFunc
		wantErr     bool
	}{
		{
			name: "successful handler",
			handlerFunc: func(client *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200}, nil
			},
			wantErr: false,
		},
		{
			name: "error handler",
			handlerFunc: func(client *http.Client, req *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			resp, err := tt.handlerFunc.Handle(client, req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, 200, resp.StatusCode)
			}
		})
	}
}

func TestSkip(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func() Handler
		wantStatus int
	}{
		{
			name: "skip middleware passes through",
			setupMock: func() Handler {
				return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200}, nil
				})
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Skip()
			handler := middleware(tt.setupMock())

			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			resp, err := handler.Handle(client, req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		middleware     Middleware
		baseHandler    Handler
		expectedCalled bool
		expectedStatus int
	}{
		{
			name: "middleware modifies request",
			middleware: func(next Handler) Handler {
				return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
					req.Header.Set("X-Custom", "test")
					return next.Handle(client, req)
				})
			},
			baseHandler: HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				assert.Equal(t, "test", req.Header.Get("X-Custom"))
				return &http.Response{StatusCode: 200}, nil
			}),
			expectedCalled: true,
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.middleware(tt.baseHandler)

			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(t, err)

			resp, err := handler.Handle(client, req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
