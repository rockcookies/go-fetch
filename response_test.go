package fetch

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse_JSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name        string
		setupResp   func() *Response
		expected    TestStruct
		expectError bool
	}{
		{
			name: "valid JSON response",
			setupResp: func() *Response {
				data := `{"name":"test","value":123}`
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(data))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected:    TestStruct{Name: "test", Value: 123},
			expectError: false,
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			defer resp.Close()

			var result TestStruct
			err := resp.JSON(&result)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.Value, result.Value)
			}
		})
	}
}

func TestResponse_XML(t *testing.T) {
	type TestStruct struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	tests := []struct {
		name        string
		setupResp   func() *Response
		expected    TestStruct
		expectError bool
	}{
		{
			name: "valid XML response",
			setupResp: func() *Response {
				data := `<test><name>test</name><value>123</value></test>`
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(data))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected:    TestStruct{Name: "test", Value: 123},
			expectError: false,
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			defer resp.Close()

			var result TestStruct
			err := resp.XML(&result)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.Value, result.Value)
			}
		})
	}
}

func TestResponse_Bytes(t *testing.T) {
	tests := []struct {
		name        string
		setupResp   func() *Response
		expected    []byte
		expectError bool
	}{
		{
			name: "simple byte response",
			setupResp: func() *Response {
				data := []byte("test data")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write(data)
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name: "empty response",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected:    nil,
			expectError: false,
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expected:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			defer resp.Close()

			result := resp.Bytes()

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResponse_String(t *testing.T) {
	tests := []struct {
		name      string
		setupResp func() *Response
		expected  string
	}{
		{
			name: "simple string response",
			setupResp: func() *Response {
				data := "hello world"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(data))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected: "hello world",
		},
		{
			name: "empty string response",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expected: "",
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			defer resp.Close()

			result := resp.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResponse_SaveToFile(t *testing.T) {
	tests := []struct {
		name        string
		setupResp   func() *Response
		fileContent string
		expectError bool
	}{
		{
			name: "save successful",
			setupResp: func() *Response {
				data := "file content"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(data))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			fileContent: "file content",
			expectError: false,
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()

			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.txt")

			err := resp.SaveToFile(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.Equal(t, tt.fileContent, string(content))
			}
		})
	}
}

func TestResponse_Read(t *testing.T) {
	tests := []struct {
		name        string
		setupResp   func() *Response
		expectError bool
		expected    string
	}{
		{
			name: "read successful",
			setupResp: func() *Response {
				data := "read data"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(data))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expectError: false,
			expected:    "read data",
		},
		{
			name: "response with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()

			buf := make([]byte, 1024)
			n, err := resp.Read(buf)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, -1, n)
			} else {
				if err != nil && err != io.EOF {
					t.Fatalf("unexpected error: %v", err)
				}
				assert.Equal(t, tt.expected, string(buf[:n]))
			}
		})
	}
}

func TestResponse_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupResp   func() *Response
		expectError bool
	}{
		{
			name: "close successful",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("data"))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
			expectError: false,
		},
		{
			name: "close with error",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("data"))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, errors.New("request error"))
			},
			expectError: true,
		},
		{
			name: "close with nil response - safe to defer",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("connection failed"))
			},
			expectError: true,
		},
		{
			name: "close with nil body - safe to defer",
			setupResp: func() *Response {
				resp := &http.Response{
					StatusCode: 200,
					Header:     http.Header{},
					Body:       nil,
				}
				return buildResponse(&http.Request{}, resp, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			err := resp.Close()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestResponse_Close_Defer verifies that defer resp.Close() is safe in all scenarios
func TestResponse_Close_Defer(t *testing.T) {
	tests := []struct {
		name      string
		setupResp func() *Response
	}{
		{
			name: "defer close on successful response",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("success"))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				return buildResponse(&http.Request{}, resp, nil)
			},
		},
		{
			name: "defer close on error response - no panic",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("network error"))
			},
		},
		{
			name: "defer close on nil body - no panic",
			setupResp: func() *Response {
				resp := &http.Response{StatusCode: 204, Body: nil}
				return buildResponse(&http.Request{}, resp, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This pattern should not panic
			func() {
				resp := tt.setupResp()
				defer resp.Close()

				// Simulate using the response
				if resp.Error == nil {
					_ = resp.String()
				}
			}()
			// If we reach here, no panic occurred
			assert.True(t, true)
		})
	}
}

func TestResponse_ClearInternalBuffer(t *testing.T) {
	tests := []struct {
		name      string
		setupResp func() *Response
	}{
		{
			name: "clear buffer with data",
			setupResp: func() *Response {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("buffer data"))
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				r := buildResponse(&http.Request{}, resp, nil)
				// Populate buffer first
				r.String()
				return r
			},
		},
		{
			name: "clear buffer with error",
			setupResp: func() *Response {
				return buildResponse(&http.Request{}, nil, errors.New("request error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()

			// Should not panic
			resp.ClearInternalBuffer()

			// After clearing, buffer should be empty if no error
			if resp.Error == nil && resp.buffer != nil {
				assert.Equal(t, 0, resp.buffer.Len())
			}
		})
	}
}

func TestBuildResponse(t *testing.T) {
	tests := []struct {
		name        string
		req         *http.Request
		resp        *http.Response
		err         error
		expectError bool
	}{
		{
			name: "successful response",
			req:  &http.Request{},
			resp: &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			},
			err:         nil,
			expectError: false,
		},
		{
			name:        "response with error",
			req:         &http.Request{},
			resp:        nil,
			err:         errors.New("test error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := buildResponse(tt.req, tt.resp, tt.err)

			assert.NotNil(t, resp)
			assert.Equal(t, tt.req, resp.RawRequest)

			if tt.expectError {
				assert.Error(t, resp.Error)
			} else {
				assert.NoError(t, resp.Error)
				assert.Equal(t, tt.resp, resp.RawResponse)
			}
		})
	}
}

func TestResponse_getInternalReader(t *testing.T) {
	tests := []struct {
		name             string
		setupResp        func() *Response
		expectBufferUsed bool
	}{
		{
			name: "empty buffer returns self",
			setupResp: func() *Response {
				resp := &Response{
					buffer: bytes.NewBuffer(nil),
					RawResponse: &http.Response{
						Body: io.NopCloser(strings.NewReader("body data")),
					},
				}
				return resp
			},
			expectBufferUsed: false,
		},
		{
			name: "populated buffer returns buffer",
			setupResp: func() *Response {
				buffer := bytes.NewBuffer([]byte("buffered data"))
				resp := &Response{
					buffer: buffer,
					RawResponse: &http.Response{
						Body: io.NopCloser(strings.NewReader("body data")),
					},
				}
				return resp
			},
			expectBufferUsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.setupResp()
			reader := resp.getInternalReader()

			assert.NotNil(t, reader)

			if tt.expectBufferUsed {
				assert.Equal(t, resp.buffer, reader)
			}
		})
	}
}
