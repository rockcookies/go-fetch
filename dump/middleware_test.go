package dump

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Logger)
	assert.Equal(t, slog.LevelInfo, opts.LogLevel)
	assert.NotNil(t, opts.LogLevelFunc)
	assert.Equal(t, int64(1024*10), opts.RequestBodyMaxSize)
	assert.Equal(t, int64(1024*100), opts.ResponseBodyMaxSize)
}

func TestDefaultOptionsLogLevelFunc(t *testing.T) {
	opts := DefaultOptions()

	tests := []struct {
		name          string
		statusCode    int
		method        string
		expectedLevel slog.Level
	}{
		{
			name:          "500 error returns Error level",
			statusCode:    500,
			method:        "GET",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "503 error returns Error level",
			statusCode:    503,
			method:        "GET",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "429 returns Info level",
			statusCode:    429,
			method:        "GET",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "404 returns Warn level",
			statusCode:    404,
			method:        "GET",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "400 returns Warn level",
			statusCode:    400,
			method:        "POST",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "OPTIONS returns Debug level",
			statusCode:    200,
			method:        "OPTIONS",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "200 GET returns Info level",
			statusCode:    200,
			method:        "GET",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "201 POST returns Info level",
			statusCode:    201,
			method:        "POST",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com/test", nil)
			level := opts.LogLevelFunc(req, tt.statusCode)
			assert.Equal(t, tt.expectedLevel, level)
		})
	}
}

func TestSkipDump(t *testing.T) {
	ctx := context.Background()
	_, ok := skipKey.GetValue(ctx)
	assert.False(t, ok)

	ctx = SkipDump(ctx)
	skip, ok := skipKey.GetValue(ctx)
	assert.True(t, ok)
	assert.True(t, skip)
}

func TestNewRoundTripper(t *testing.T) {
	opts := DefaultOptions()
	optionsFunc := func(req *http.Request) *Options {
		return opts
	}

	rt := NewRoundTripper(nil, optionsFunc)
	assert.NotNil(t, rt)
	assert.NotNil(t, rt.next)
	assert.NotNil(t, rt.optionsFunc)

	// Test with custom transport
	customTransport := http.DefaultTransport
	rt = NewRoundTripper(customTransport, optionsFunc)
	assert.Equal(t, customTransport, rt.next)
}

func TestNewRoundTripperWithOptions(t *testing.T) {
	opts := DefaultOptions()
	rt := NewRoundTripperWithOptions(nil, opts)

	assert.NotNil(t, rt)
	assert.NotNil(t, rt.next)
	assert.NotNil(t, rt.optionsFunc)

	// Verify options function returns the correct options
	req := httptest.NewRequest("GET", "http://example.com", nil)
	returnedOpts := rt.optionsFunc(req)
	assert.Equal(t, opts, returnedOpts)
}

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestRoundTripperWithSkip(t *testing.T) {
	// Create a mock transport
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("test response")),
		Header:     http.Header{},
	}
	transport := &mockTransport{response: mockResp}

	// Create RoundTripper
	opts := DefaultOptions()
	rt := NewRoundTripperWithOptions(transport, opts)

	// Create request with SkipDump context
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req = req.WithContext(SkipDump(req.Context()))

	// Execute
	resp, err := rt.RoundTrip(req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRoundTripperBasicRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))
	defer server.Close()

	// Capture logs
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	opts := DefaultOptions()
	opts.Logger = logger
	opts.RequestBodyFilter = func(req *http.Request) bool { return true }
	opts.ResponseBodyFilter = func(req *http.Request) bool { return true }

	rt := NewRoundTripperWithOptions(http.DefaultTransport, opts)

	// Create request
	req := httptest.NewRequest("GET", server.URL, nil)

	// Execute
	resp, err := rt.RoundTrip(req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify log was written
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "HTTP request completed")
}

func TestRoundTripperWithRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "request payload", string(body))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	opts := DefaultOptions()
	opts.RequestBodyFilter = func(req *http.Request) bool { return true }
	opts.RequestBodyMaxSize = 1024

	rt := NewRoundTripperWithOptions(http.DefaultTransport, opts)

	reqBody := strings.NewReader("request payload")
	req := httptest.NewRequest("POST", server.URL, reqBody)

	resp, err := rt.RoundTrip(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestRoundTripperWithResponseBody(t *testing.T) {
	responseBody := "test response body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()

	opts := DefaultOptions()
	opts.ResponseBodyFilter = func(req *http.Request) bool { return true }
	opts.ResponseBodyMaxSize = 1024

	rt := NewRoundTripperWithOptions(http.DefaultTransport, opts)

	req := httptest.NewRequest("GET", server.URL, nil)

	resp, err := rt.RoundTrip(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify response body can still be read
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, responseBody, string(body))
}

func TestRoundTripperWithFilters(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name      string
		filters   []Filter
		method    string
		status    int
		shouldLog bool
	}{
		{
			name:      "filter accepts request",
			filters:   []Filter{AcceptMethod("GET")},
			method:    "GET",
			status:    200,
			shouldLog: true,
		},
		{
			name:      "filter rejects request",
			filters:   []Filter{AcceptMethod("POST")},
			method:    "GET",
			status:    200,
			shouldLog: false,
		},
		{
			name:      "multiple filters all pass",
			filters:   []Filter{AcceptMethod("GET"), AcceptStatusGreaterThanOrEqual(200)},
			method:    "GET",
			status:    200,
			shouldLog: true,
		},
		{
			name:      "multiple filters one fails",
			filters:   []Filter{AcceptMethod("GET"), AcceptStatus(404)},
			method:    "GET",
			status:    200,
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			opts := DefaultOptions()
			opts.Logger = logger
			opts.Filters = tt.filters

			rt := NewRoundTripperWithOptions(http.DefaultTransport, opts)

			req := httptest.NewRequest(tt.method, server.URL, nil)
			resp, err := rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)

			logOutput := logBuf.String()
			if tt.shouldLog {
				assert.Contains(t, logOutput, "HTTP request completed")
			} else {
				assert.Empty(t, logOutput)
			}
		})
	}
}

func TestRoundTripperWithError(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	expectedErr := errors.New("transport error")
	transport := &mockTransport{err: expectedErr}

	opts := DefaultOptions()
	opts.Logger = logger

	rt := NewRoundTripperWithOptions(transport, opts)

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resp)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "transport error")
}

func TestRoundTripperCustomAttributes(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := DefaultOptions()
	opts.Logger = logger
	opts.RequestAttrs = func(req *http.Request) []slog.Attr {
		return []slog.Attr{
			slog.String("custom_request", "value"),
		}
	}
	opts.ResponseAttrs = func(resp *http.Response, duration time.Duration) []slog.Attr {
		return []slog.Attr{
			slog.String("custom_response", "value"),
		}
	}

	rt := NewRoundTripperWithOptions(http.DefaultTransport, opts)

	req := httptest.NewRequest("GET", server.URL, nil)
	resp, err := rt.RoundTrip(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "custom_request")
	assert.Contains(t, logOutput, "custom_response")
}

func TestGetDrainedBodyAttrs(t *testing.T) {
	tests := []struct {
		name     string
		body     *drainedBody
		expected int
	}{
		{
			name:     "nil body",
			body:     nil,
			expected: 0,
		},
		{
			name: "normal body",
			body: &drainedBody{
				body:      bytes.NewBufferString("test content"),
				size:      12,
				truncated: false,
			},
			expected: 2,
		},
		{
			name: "truncated body",
			body: &drainedBody{
				body:      bytes.NewBufferString("test"),
				size:      100,
				truncated: true,
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := getDrainedBodyAttrs(tt.body)
			assert.Equal(t, tt.expected, len(attrs))

			if tt.body != nil {
				assert.Contains(t, attrs, slog.String("content", tt.body.body.String()))
				assert.Contains(t, attrs, slog.Int64("size", tt.body.size))

				if tt.body.truncated {
					assert.Contains(t, attrs, slog.Bool("truncated", true))
				}
			}
		})
	}
}

func TestGetHeaderAttrs(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		filter   func(key string, value []string) []any
		expected int
	}{
		{
			name:     "empty header",
			header:   http.Header{},
			filter:   nil,
			expected: 0,
		},
		{
			name: "single value header",
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			filter:   nil,
			expected: 1,
		},
		{
			name: "multi value header",
			header: http.Header{
				"Accept": []string{"application/json", "text/html"},
			},
			filter:   nil,
			expected: 1,
		},
		{
			name: "with filter",
			header: http.Header{
				"Authorization": []string{"Bearer token"},
				"Content-Type":  []string{"application/json"},
			},
			filter: func(key string, value []string) []any {
				if key == "Authorization" {
					return []any{slog.String(key, "[REDACTED]")}
				}
				return []any{slog.String(key, value[0])}
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := getHeaderAttrs(tt.header, tt.filter)
			assert.Equal(t, tt.expected, len(attrs))
		})
	}
}

func TestBuildLogMessage(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		expected string
	}{
		{
			name:     "with error",
			resp:     nil,
			err:      errors.New("test error"),
			expected: "HTTP request failed",
		},
		{
			name:     "without response",
			resp:     nil,
			err:      nil,
			expected: "HTTP request completed",
		},
		{
			name: "with response",
			resp: &http.Response{
				StatusCode: 200,
			},
			err:      nil,
			expected: "HTTP request completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildLogMessage(tt.resp, tt.err)
			assert.Equal(t, tt.expected, msg)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{
			name:     "microseconds",
			duration: 123 * time.Microsecond,
			contains: "123",
		},
		{
			name:     "milliseconds",
			duration: 456 * time.Millisecond,
			contains: "ms",
		},
		{
			name:     "seconds",
			duration: 2 * time.Second,
			contains: "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestScheme(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		expected string
	}{
		{
			name: "http request",
			req: &http.Request{
				TLS: nil,
			},
			expected: "http",
		},
		{
			name: "https request",
			req: &http.Request{
				TLS: &tls.ConnectionState{},
			},
			expected: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheme(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestURL(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		expected string
	}{
		{
			name: "http url",
			req: &http.Request{
				Host: "example.com",
				URL: &url.URL{
					Path: "/api/users",
				},
				TLS: nil,
			},
			expected: "http://example.com/api/users",
		},
		{
			name: "https url",
			req: &http.Request{
				Host: "api.example.com",
				URL: &url.URL{
					Path: "/v1/users",
				},
				TLS: &tls.ConnectionState{},
			},
			expected: "https://api.example.com/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := requestURL(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}
