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
)

func TestSetBody(t *testing.T) {
	tests := []struct {
		name            string
		reader          io.Reader
		expectedBody    string
		expectedLength  int64
		shouldSetLength bool
	}{
		{
			name:            "set body with bytes.Buffer",
			reader:          bytes.NewBufferString("test body"),
			expectedBody:    "test body",
			expectedLength:  9,
			shouldSetLength: true,
		},
		{
			name:            "set body with strings.Reader",
			reader:          strings.NewReader("hello world"),
			expectedBody:    "hello world",
			expectedLength:  11,
			shouldSetLength: true,
		},
		{
			name:            "nil reader",
			reader:          nil,
			expectedBody:    "",
			expectedLength:  0,
			shouldSetLength: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBody(tt.reader)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if tt.reader == nil {
					if req.Body != nil && req.Body != http.NoBody {
						t.Error("expected nil body")
					}
					return nil, nil
				}

				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				if string(body) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(body))
				}

				if tt.shouldSetLength && req.ContentLength != tt.expectedLength {
					t.Errorf("expected ContentLength %d, got %d", tt.expectedLength, req.ContentLength)
				}

				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetBodyGet(t *testing.T) {
	tests := []struct {
		name         string
		getReader    func() (io.Reader, error)
		expectedBody string
		shouldError  bool
	}{
		{
			name: "get body successfully",
			getReader: func() (io.Reader, error) {
				return strings.NewReader("lazy body"), nil
			},
			expectedBody: "lazy body",
			shouldError:  false,
		},
		{
			name: "get body returns error",
			getReader: func() (io.Reader, error) {
				return nil, io.ErrUnexpectedEOF
			},
			shouldError: true,
		},
		{
			name:         "nil getter",
			getReader:    nil,
			expectedBody: "",
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBodyGet(tt.getReader)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if tt.getReader == nil {
					if req.GetBody != nil {
						t.Error("expected nil GetBody")
					}
					return nil, nil
				}

				if req.GetBody == nil {
					t.Fatal("expected GetBody to be set")
				}

				body, err := req.GetBody()
				if tt.shouldError {
					if err == nil {
						t.Error("expected error from GetBody")
					}
					return nil, nil
				}

				if err != nil {
					t.Fatalf("GetBody returned error: %v", err)
				}

				data, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				if string(data) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(data))
				}

				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetBodyGetBytes(t *testing.T) {
	tests := []struct {
		name           string
		getBytes       func() ([]byte, error)
		expectedBody   string
		expectedLength int64
		shouldError    bool
	}{
		{
			name: "get bytes successfully",
			getBytes: func() ([]byte, error) {
				return []byte("byte body"), nil
			},
			expectedBody:   "byte body",
			expectedLength: 9,
			shouldError:    false,
		},
		{
			name: "get bytes returns error",
			getBytes: func() ([]byte, error) {
				return nil, io.ErrUnexpectedEOF
			},
			shouldError: true,
		},
		{
			name:        "nil getter",
			getBytes:    nil,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBodyGetBytes(tt.getBytes)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				if tt.shouldError {
					// The error should have been returned during getBytes call
					return nil, nil
				}

				if tt.getBytes == nil {
					return nil, nil
				}

				if req.GetBody == nil {
					t.Fatal("expected GetBody to be set")
				}

				if req.ContentLength != tt.expectedLength {
					t.Errorf("expected ContentLength %d, got %d", tt.expectedLength, req.ContentLength)
				}

				body, err := req.GetBody()
				if err != nil {
					t.Fatalf("GetBody returned error: %v", err)
				}

				data, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				if string(data) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(data))
				}

				return nil, nil
			}))

			_, handlerErr := handler.Handle(&http.Client{}, req)
			if tt.shouldError && handlerErr == nil {
				t.Error("expected error from handler")
			}
		})
	}
}

func TestSetBodyJSON(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name         string
		data         any
		expectedBody string
		wantJSON     bool
	}{
		{
			name:         "string data",
			data:         "plain string",
			expectedBody: "plain string",
			wantJSON:     false,
		},
		{
			name:         "byte slice data",
			data:         []byte("byte data"),
			expectedBody: "byte data",
			wantJSON:     false,
		},
		{
			name: "struct data",
			data: testStruct{
				Name:  "test",
				Value: 123,
			},
			expectedBody: `{"name":"test","value":123}`,
			wantJSON:     true,
		},
		{
			name: "map data",
			data: map[string]string{
				"key": "value",
			},
			expectedBody: `{"key":"value"}`,
			wantJSON:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBodyJSON(tt.data)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Check Content-Type header
				if ct := req.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
				}

				if req.GetBody == nil {
					t.Fatal("expected GetBody to be set")
				}

				body, err := req.GetBody()
				if err != nil {
					t.Fatalf("GetBody returned error: %v", err)
				}

				data, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				bodyStr := strings.TrimSpace(string(data))
				if tt.wantJSON {
					// For JSON, we need to unmarshal and compare to handle formatting
					var got, want interface{}
					if err := json.Unmarshal([]byte(bodyStr), &got); err != nil {
						t.Fatalf("failed to unmarshal got: %v", err)
					}
					if err := json.Unmarshal([]byte(tt.expectedBody), &want); err != nil {
						t.Fatalf("failed to unmarshal want: %v", err)
					}
					// Simple comparison, could use reflect.DeepEqual for complex cases
					gotJSON, _ := json.Marshal(got)
					wantJSON, _ := json.Marshal(want)
					if string(gotJSON) != string(wantJSON) {
						t.Errorf("expected JSON %q, got %q", string(wantJSON), string(gotJSON))
					}
				} else {
					if bodyStr != tt.expectedBody {
						t.Errorf("expected body %q, got %q", tt.expectedBody, bodyStr)
					}
				}

				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetBodyXML(t *testing.T) {
	type testStruct struct {
		XMLName xml.Name `xml:"root"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	tests := []struct {
		name         string
		data         any
		expectedBody string
		wantXML      bool
	}{
		{
			name:         "string data",
			data:         "plain string",
			expectedBody: "plain string",
			wantXML:      false,
		},
		{
			name:         "byte slice data",
			data:         []byte("byte data"),
			expectedBody: "byte data",
			wantXML:      false,
		},
		{
			name: "struct data",
			data: testStruct{
				Name:  "test",
				Value: 123,
			},
			wantXML: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBodyXML(tt.data)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Check Content-Type header
				if ct := req.Header.Get("Content-Type"); ct != "application/xml" {
					t.Errorf("expected Content-Type %q, got %q", "application/xml", ct)
				}

				if req.GetBody == nil {
					t.Fatal("expected GetBody to be set")
				}

				body, err := req.GetBody()
				if err != nil {
					t.Fatalf("GetBody returned error: %v", err)
				}

				data, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				bodyStr := strings.TrimSpace(string(data))
				if tt.wantXML {
					// Just verify it's valid XML
					if !strings.HasPrefix(bodyStr, "<") {
						t.Errorf("expected XML body to start with <, got %q", bodyStr)
					}
				} else {
					if bodyStr != tt.expectedBody {
						t.Errorf("expected body %q, got %q", tt.expectedBody, bodyStr)
					}
				}

				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}

func TestSetBodyForm(t *testing.T) {
	tests := []struct {
		name         string
		data         url.Values
		expectedBody string
	}{
		{
			name: "single form field",
			data: url.Values{
				"key": []string{"value"},
			},
			expectedBody: "key=value",
		},
		{
			name: "multiple form fields",
			data: url.Values{
				"key1": []string{"value1"},
				"key2": []string{"value2"},
			},
		},
		{
			name: "field with multiple values",
			data: url.Values{
				"key": []string{"value1", "value2"},
			},
		},
		{
			name:         "empty form",
			data:         url.Values{},
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/path", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			middleware := SetBodyForm(tt.data)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Check Content-Type header
				if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
					t.Errorf("expected Content-Type %q, got %q", "application/x-www-form-urlencoded", ct)
				}

				if req.GetBody == nil {
					t.Fatal("expected GetBody to be set")
				}

				body, err := req.GetBody()
				if err != nil {
					t.Fatalf("GetBody returned error: %v", err)
				}

				data, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}

				// Parse the form data back
				parsedData, err := url.ParseQuery(string(data))
				if err != nil {
					t.Fatalf("failed to parse form data: %v", err)
				}

				// Compare the parsed data with the original
				for key, values := range tt.data {
					parsedValues := parsedData[key]
					if len(parsedValues) != len(values) {
						t.Errorf("expected %d values for key %q, got %d", len(values), key, len(parsedValues))
					}
					for i, v := range values {
						if parsedValues[i] != v {
							t.Errorf("expected value[%d] %q for key %q, got %q", i, v, key, parsedValues[i])
						}
					}
				}

				return nil, nil
			}))

			handler.Handle(&http.Client{}, req)
		})
	}
}
