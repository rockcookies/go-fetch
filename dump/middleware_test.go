package dump

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper is a test RoundTripper that returns predefined responses.
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewRoundTripper(t *testing.T) {
	tests := []struct {
		name     string
		next     http.RoundTripper
		wantNext http.RoundTripper
	}{
		{
			name:     "with custom transport",
			next:     &mockRoundTripper{},
			wantNext: &mockRoundTripper{},
		},
		{
			name:     "with nil transport uses default",
			next:     nil,
			wantNext: http.DefaultTransport,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRoundTripper(tt.next, func(req *http.Request) *Options {
				return DefaultOptions()
			})

			require.NotNil(t, rt)
			assert.Equal(t, tt.wantNext, rt.next)
			assert.NotNil(t, rt.optionsFunc)
		})
	}
}

func TestNewRoundTripperWithOptions(t *testing.T) {
	opts := DefaultOptions()
	rt := NewRoundTripperWithOptions(nil, opts)

	require.NotNil(t, rt)
	assert.NotNil(t, rt.optionsFunc)

	// Verify optionsFunc returns the provided options
	req := httptest.NewRequest("GET", "http://example.com", nil)
	gotOpts := rt.optionsFunc(req)
	assert.Equal(t, opts, gotOpts)
}

func TestRoundTripper_RoundTrip_BasicRequest(t *testing.T) {
	// Test basic request logging without body
	mockResp := &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(`{"ok":true}`)),
		ContentLength: 11,
	}

	mock := &mockRoundTripper{response: mockResp}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := &Options{
		Logger:   logger,
		LogLevel: slog.LevelInfo,
	}

	rt := NewRoundTripperWithOptions(mock, opts)
	req := httptest.NewRequest("GET", "http://example.com/api/test?foo=bar", nil)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/api/test")
	assert.Contains(t, logOutput, "foo=bar")
	assert.Contains(t, logOutput, "200")
}

func TestRoundTripper_RoundTrip_WithRequestBody(t *testing.T) {
	mockResp := &http.Response{
		StatusCode: 201,
		Status:     "201 Created",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	mock := &mockRoundTripper{response: mockResp}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := &Options{
		Logger: logger,
		RequestBodyFilters: []func(req *http.Request) bool{
			func(req *http.Request) bool { return true },
		},
		RequestBodyMaxSize: 1024,
	}

	rt := NewRoundTripperWithOptions(mock, opts)

	requestBody := `{"name":"test","value":42}`
	req := httptest.NewRequest("POST", "http://example.com/api/create", strings.NewReader(requestBody))

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 201, resp.StatusCode)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "POST")
	assert.Contains(t, logOutput, "name")
	assert.Contains(t, logOutput, "test")
	assert.Contains(t, logOutput, "value")
	assert.Contains(t, logOutput, "42")
}

func TestRoundTripper_RoundTrip_WithResponseBody(t *testing.T) {
	responseBody := `{"id":123,"status":"created"}`
	mockResp := &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(strings.NewReader(responseBody)),
		ContentLength: int64(len(responseBody)),
	}

	mock := &mockRoundTripper{response: mockResp}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := &Options{
		Logger: logger,
		RequestBodyFilters: []func(req *http.Request) bool{
			func(req *http.Request) bool { return true },
		},
		ResponseBodyFilters: []func(req *http.Request, resp *http.Response, err error) bool{
			func(req *http.Request, resp *http.Response, err error) bool { return true },
		},
		RequestBodyMaxSize:  1024,
		ResponseBodyMaxSize: 1024,
	}

	rt := NewRoundTripperWithOptions(mock, opts)
	req := httptest.NewRequest("GET", "http://example.com/api/resource", nil)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "id")
	assert.Contains(t, logOutput, "123")
	assert.Contains(t, logOutput, "status")
	assert.Contains(t, logOutput, "created")

	// Verify response body is still readable
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, responseBody, string(body))
}

func TestRoundTripper_RoundTrip_Filters(t *testing.T) {
	tests := []struct {
		name          string
		filters       []func(req *http.Request) bool
		shouldExecute bool
	}{
		{
			name: "no filters executes",
			filters: []func(req *http.Request) bool{
				func(req *http.Request) bool { return true },
			},
			shouldExecute: true,
		},
		{
			name: "filter returns false skips logging",
			filters: []func(req *http.Request) bool{
				func(req *http.Request) bool { return false },
			},
			shouldExecute: false,
		},
		{
			name: "multiple filters all must pass",
			filters: []func(req *http.Request) bool{
				func(req *http.Request) bool { return true },
				func(req *http.Request) bool { return true },
			},
			shouldExecute: true,
		},
		{
			name: "one filter fails stops execution",
			filters: []func(req *http.Request) bool{
				func(req *http.Request) bool { return true },
				func(req *http.Request) bool { return false },
			},
			shouldExecute: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResp := &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       io.NopCloser(strings.NewReader("")),
			}

			mock := &mockRoundTripper{response: mockResp}

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			opts := &Options{
				Logger:  logger,
				Filters: tt.filters,
			}

			rt := NewRoundTripperWithOptions(mock, opts)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			logOutput := logBuf.String()
			if tt.shouldExecute {
				assert.NotEmpty(t, logOutput)
			} else {
				assert.Empty(t, logOutput)
			}
		})
	}
}

func TestRoundTripper_RoundTrip_LogLevelFunc(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel slog.Level
		shouldLog     bool
		logHandlerLvl slog.Level
	}{
		{
			name:          "500 error logs at error level",
			statusCode:    500,
			expectedLevel: slog.LevelError,
			shouldLog:     true,
			logHandlerLvl: slog.LevelInfo,
		},
		{
			name:          "404 logs at warn level",
			statusCode:    404,
			expectedLevel: slog.LevelWarn,
			shouldLog:     true,
			logHandlerLvl: slog.LevelInfo,
		},
		{
			name:          "200 logs at info level",
			statusCode:    200,
			expectedLevel: slog.LevelInfo,
			shouldLog:     true,
			logHandlerLvl: slog.LevelInfo,
		},
		{
			name:          "200 skipped at warn handler level",
			statusCode:    200,
			expectedLevel: slog.LevelInfo,
			shouldLog:     false,
			logHandlerLvl: slog.LevelWarn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResp := &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       io.NopCloser(strings.NewReader("")),
			}

			mock := &mockRoundTripper{response: mockResp}

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: tt.logHandlerLvl}))

			opts := &Options{
				Logger:       logger,
				LogLevelFunc: DefaultLogLevelFunc,
			}

			rt := NewRoundTripperWithOptions(mock, opts)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			logOutput := logBuf.String()
			if tt.shouldLog {
				assert.NotEmpty(t, logOutput, "Expected log output but got none")
			} else {
				assert.Empty(t, logOutput, "Expected no log output but got: %s", logOutput)
			}
		})
	}
}

func TestRoundTripper_RoundTrip_Error(t *testing.T) {
	mockErr := io.EOF

	mock := &mockRoundTripper{err: mockErr}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := &Options{
		Logger: logger,
	}

	rt := NewRoundTripperWithOptions(mock, opts)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	resp, err := rt.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "EOF")
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	require.NotNil(t, opts)
	assert.NotNil(t, opts.Logger)
	assert.Equal(t, slog.LevelInfo, opts.LogLevel)
	assert.NotNil(t, opts.LogLevelFunc)
	assert.Equal(t, int64(1024*10), opts.RequestBodyMaxSize)
	assert.Equal(t, int64(1024*100), opts.ResponseBodyMaxSize)
}

func TestDefaultLogLevelFunc(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		method     string
		expected   slog.Level
	}{
		{
			name:       "500 server error",
			statusCode: 500,
			method:     "GET",
			expected:   slog.LevelError,
		},
		{
			name:       "503 service unavailable",
			statusCode: 503,
			method:     "POST",
			expected:   slog.LevelError,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			method:     "GET",
			expected:   slog.LevelInfo,
		},
		{
			name:       "404 not found",
			statusCode: 404,
			method:     "GET",
			expected:   slog.LevelWarn,
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			method:     "POST",
			expected:   slog.LevelWarn,
		},
		{
			name:       "OPTIONS preflight",
			statusCode: 200,
			method:     "OPTIONS",
			expected:   slog.LevelDebug,
		},
		{
			name:       "200 success",
			statusCode: 200,
			method:     "GET",
			expected:   slog.LevelInfo,
		},
		{
			name:       "201 created",
			statusCode: 201,
			method:     "POST",
			expected:   slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com", nil)
			resp := &http.Response{StatusCode: tt.statusCode}

			level := DefaultLogLevelFunc(req, resp, nil)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		contains string
	}{
		{
			name:     "microseconds",
			duration: "123µs",
			contains: "µs",
		},
		{
			name:     "milliseconds",
			duration: "5ms",
			contains: "ms",
		},
		{
			name:     "seconds",
			duration: "2s",
			contains: "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatDuration(mustParseDuration(tt.duration))
			assert.Contains(t, formatted, tt.contains)
		})
	}
}

func TestGetDrainedBodyAttrs(t *testing.T) {
	tests := []struct {
		name         string
		drainedBody  *drainedBody
		expectEmpty  bool
		expectSize   int64
		expectTrunc  bool
		expectFields []string
	}{
		{
			name:        "nil body returns empty",
			drainedBody: nil,
			expectEmpty: true,
		},
		{
			name: "normal body",
			drainedBody: &drainedBody{
				body:      bytes.NewBufferString("test content"),
				size:      12,
				truncated: false,
			},
			expectEmpty:  false,
			expectSize:   12,
			expectTrunc:  false,
			expectFields: []string{"content", "size"},
		},
		{
			name: "truncated body",
			drainedBody: &drainedBody{
				body:      bytes.NewBufferString("trunca"),
				size:      100,
				truncated: true,
			},
			expectEmpty:  false,
			expectSize:   100,
			expectTrunc:  true,
			expectFields: []string{"content", "size", "truncated", "captured_size"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := getDrainedBodyAttrs(tt.drainedBody)

			if tt.expectEmpty {
				assert.Empty(t, attrs)
				return
			}

			assert.NotEmpty(t, attrs)
			// Verify expected fields are present
			attrMap := attrsToMap(attrs)
			for _, field := range tt.expectFields {
				_, exists := attrMap[field]
				assert.True(t, exists, "Expected field %s not found", field)
			}
		})
	}
}

func TestGetHeaderAttrs(t *testing.T) {
	tests := []struct {
		name         string
		header       http.Header
		filter       func(key string, value []string) []any
		expectCount  int
		expectFields []string
	}{
		{
			name: "single value headers",
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"Accept":       []string{"*/*"},
			},
			filter:       nil,
			expectCount:  2,
			expectFields: []string{"Content-Type", "Accept"},
		},
		{
			name: "multi value header",
			header: http.Header{
				"Set-Cookie": []string{"session=abc", "token=xyz"},
			},
			filter:       nil,
			expectCount:  1,
			expectFields: []string{"Set-Cookie"},
		},
		{
			name: "filtered headers",
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
			expectCount:  2,
			expectFields: []string{"Authorization", "Content-Type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := getHeaderAttrs(tt.header, tt.filter)
			attrsMap := attrsToMap(attrs)
			assert.Len(t, attrsMap, tt.expectCount)
		})
	}
}

// Helper functions

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func attrsToMap(attrs []any) map[string]any {
	m := make(map[string]any)
	for _, attr := range attrs {
		if a, ok := attr.(slog.Attr); ok {
			m[a.Key] = a.Value.Any()
		}
	}
	return m
}
