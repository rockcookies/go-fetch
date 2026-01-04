package fetch

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyReader(t *testing.T) {
	tests := []struct {
		name                string
		reader              io.Reader
		options             []func(*BodyOptions)
		expectedContentType string
		expectedContentLen  int64
		validateBody        func(t *testing.T, body io.ReadCloser)
	}{
		{
			name:                "simple string reader",
			reader:              strings.NewReader("test body"),
			options:             []func(*BodyOptions){},
			expectedContentType: "",
			expectedContentLen:  0,
			validateBody: func(t *testing.T, body io.ReadCloser) {
				data, err := io.ReadAll(body)
				require.NoError(t, err)
				assert.Equal(t, "test body", string(data))
			},
		},
		{
			name:   "with content type",
			reader: strings.NewReader("json data"),
			options: []func(*BodyOptions){
				func(o *BodyOptions) { o.ContentType = "application/json" },
			},
			expectedContentType: "application/json",
		},
		{
			name:   "auto set content length",
			reader: bytes.NewReader([]byte("test")),
			options: []func(*BodyOptions){
				func(o *BodyOptions) { o.AutoSetContentLength = true },
			},
			expectedContentLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := BodyReader(tt.reader, tt.options...)

			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if tt.expectedContentType != "" {
					assert.Equal(t, tt.expectedContentType, req.Header.Get("Content-Type"))
				}
				if tt.expectedContentLen > 0 {
					assert.Equal(t, tt.expectedContentLen, req.ContentLength)
				}
				if tt.validateBody != nil {
					tt.validateBody(t, req.Body)
				}
				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("POST", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestBodyJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name                string
		data                any
		expectedContentType string
		validateBody        func(t *testing.T, body []byte)
	}{
		{
			name:                "struct to JSON",
			data:                TestStruct{Name: "test", Value: 123},
			expectedContentType: "application/json",
			validateBody: func(t *testing.T, body []byte) {
				var result TestStruct
				err := json.Unmarshal(body, &result)
				require.NoError(t, err)
				assert.Equal(t, "test", result.Name)
				assert.Equal(t, 123, result.Value)
			},
		},
		{
			name:                "string data",
			data:                "plain text",
			expectedContentType: "application/json",
			validateBody: func(t *testing.T, body []byte) {
				assert.Equal(t, "plain text", string(body))
			},
		},
		{
			name:                "byte slice",
			data:                []byte("byte data"),
			expectedContentType: "application/json",
			validateBody: func(t *testing.T, body []byte) {
				assert.Equal(t, "byte data", string(body))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := BodyJSON(tt.data)

			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				assert.Equal(t, tt.expectedContentType, req.Header.Get("Content-Type"))

				if req.GetBody != nil {
					body, err := req.GetBody()
					require.NoError(t, err)
					data, err := io.ReadAll(body)
					require.NoError(t, err)
					if tt.validateBody != nil {
						tt.validateBody(t, data)
					}
				}

				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("POST", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestBodyXML(t *testing.T) {
	type TestStruct struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	tests := []struct {
		name                string
		data                any
		expectedContentType string
		validateBody        func(t *testing.T, body []byte)
	}{
		{
			name:                "struct to XML",
			data:                TestStruct{Name: "test", Value: 123},
			expectedContentType: "application/xml",
			validateBody: func(t *testing.T, body []byte) {
				var result TestStruct
				err := xml.Unmarshal(body, &result)
				require.NoError(t, err)
				assert.Equal(t, "test", result.Name)
				assert.Equal(t, 123, result.Value)
			},
		},
		{
			name:                "string data",
			data:                "<test>xml string</test>",
			expectedContentType: "application/xml",
			validateBody: func(t *testing.T, body []byte) {
				assert.Equal(t, "<test>xml string</test>", string(body))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := BodyXML(tt.data)

			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				assert.Equal(t, tt.expectedContentType, req.Header.Get("Content-Type"))

				if req.GetBody != nil {
					body, err := req.GetBody()
					require.NoError(t, err)
					data, err := io.ReadAll(body)
					require.NoError(t, err)
					if tt.validateBody != nil {
						tt.validateBody(t, data)
					}
				}

				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("POST", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestBodyForm(t *testing.T) {
	tests := []struct {
		name                string
		data                url.Values
		expectedContentType string
		expectedBody        string
	}{
		{
			name: "simple form",
			data: url.Values{
				"username": []string{"john"},
				"password": []string{"secret"},
			},
			expectedContentType: "application/x-www-form-urlencoded",
		},
		{
			name:                "empty form",
			data:                url.Values{},
			expectedContentType: "application/x-www-form-urlencoded",
			expectedBody:        "",
		},
		{
			name: "multiple values",
			data: url.Values{
				"tags": []string{"go", "http", "testing"},
			},
			expectedContentType: "application/x-www-form-urlencoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := BodyForm(tt.data)

			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				assert.Equal(t, tt.expectedContentType, req.Header.Get("Content-Type"))

				if req.GetBody != nil {
					body, err := req.GetBody()
					require.NoError(t, err)
					data, err := io.ReadAll(body)
					require.NoError(t, err)

					if tt.expectedBody != "" {
						assert.Equal(t, tt.expectedBody, string(data))
					} else if len(tt.data) > 0 {
						// Verify it's valid form data
						assert.Contains(t, string(data), "=")
					}
				}

				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("POST", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}

func TestBodyGetReader(t *testing.T) {
	tests := []struct {
		name         string
		getReader    func() (io.Reader, error)
		options      []func(*BodyOptions)
		expectError  bool
		validateBody func(t *testing.T, body io.ReadCloser)
	}{
		{
			name: "successful reader",
			getReader: func() (io.Reader, error) {
				return strings.NewReader("dynamic content"), nil
			},
			expectError: false,
			validateBody: func(t *testing.T, body io.ReadCloser) {
				data, err := io.ReadAll(body)
				require.NoError(t, err)
				assert.Equal(t, "dynamic content", string(data))
			},
		},
		{
			name: "reader with error",
			getReader: func() (io.Reader, error) {
				return nil, assert.AnError
			},
			expectError: true,
		},
		{
			name: "with content type",
			getReader: func() (io.Reader, error) {
				return strings.NewReader("text"), nil
			},
			options: []func(*BodyOptions){
				func(o *BodyOptions) { o.ContentType = "text/plain" },
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := BodyGetReader(tt.getReader, tt.options...)

			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if req.GetBody != nil && !tt.expectError {
					body, err := req.GetBody()
					if tt.expectError {
						assert.Error(t, err)
					} else {
						require.NoError(t, err)
						if tt.validateBody != nil {
							tt.validateBody(t, body)
						}
					}
				}
				return &http.Response{StatusCode: 200}, nil
			}))

			req, err := http.NewRequest("POST", "http://example.com", nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)
			assert.NoError(t, err)
		})
	}
}
