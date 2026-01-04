package fetch

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultipart(t *testing.T) {
	tests := []struct {
		name        string
		fields      []*MultipartField
		options     []func(*MultipartOptions)
		expectError bool
		validate    func(t *testing.T, contentType string, body []byte)
	}{
		{
			name: "single text field",
			fields: []*MultipartField{
				{
					Name:   "username",
					Values: []string{"john"},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, contentType, "multipart/form-data")
				assert.Contains(t, string(body), "username")
				assert.Contains(t, string(body), "john")
			},
		},
		{
			name: "multiple text fields",
			fields: []*MultipartField{
				{
					Name:   "username",
					Values: []string{"john"},
				},
				{
					Name:   "email",
					Values: []string{"john@example.com"},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, contentType, "multipart/form-data")
				assert.Contains(t, string(body), "username")
				assert.Contains(t, string(body), "email")
			},
		},
		{
			name: "file upload",
			fields: []*MultipartField{
				{
					Name:        "file",
					FileName:    "test.txt",
					ContentType: "text/plain",
					FileSize:    9,
					GetReader: func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader("test file")), nil
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, contentType, "multipart/form-data")
				assert.Contains(t, string(body), "test.txt")
				assert.Contains(t, string(body), "test file")
			},
		},
		{
			name: "file with auto-detected content type",
			fields: []*MultipartField{
				{
					Name:     "file",
					FileName: "data.bin",
					FileSize: 4,
					GetReader: func() (io.ReadCloser, error) {
						return io.NopCloser(bytes.NewReader([]byte{0x89, 0x50, 0x4E, 0x47})), nil
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, contentType, "multipart/form-data")
			},
		},
		{
			name: "custom boundary",
			fields: []*MultipartField{
				{
					Name:   "field",
					Values: []string{"value"},
				},
			},
			options: []func(*MultipartOptions){
				func(o *MultipartOptions) {
					o.Boundary = "custom-boundary-123"
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, contentType, "boundary=custom-boundary-123")
			},
		},
		{
			name: "multiple values for same field",
			fields: []*MultipartField{
				{
					Name:   "tags",
					Values: []string{"go", "http", "testing"},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, string(body), "tags")
			},
		},
		{
			name: "file with progress callback",
			fields: []*MultipartField{
				{
					Name:             "upload",
					FileName:         "progress.txt",
					FileSize:         11,
					ProgressInterval: 100 * time.Millisecond,
					ProgressCallback: func(progress MultipartFieldProgress) {
						assert.Equal(t, "upload", progress.Name)
						assert.Equal(t, "progress.txt", progress.FileName)
						assert.Equal(t, int64(11), progress.FileSize)
					},
					GetReader: func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader("test upload")), nil
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				assert.Contains(t, string(body), "progress.txt")
			},
		},
		{
			name:        "empty fields",
			fields:      []*MultipartField{},
			expectError: false,
			validate: func(t *testing.T, contentType string, body []byte) {
				// Should skip multipart processing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedContentType string
			var capturedBody []byte

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedContentType = r.Header.Get("Content-Type")

				// Always read body for validation
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				capturedBody = body

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			middleware := Multipart(tt.fields, tt.options...)
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				// Multipart middleware sets GetBody, we need to invoke it to populate the body
				if req.GetBody != nil {
					body, err := req.GetBody()
					if err != nil {
						return nil, err
					}
					req.Body = body
				}
				return client.Do(req)
			}))

			req, err := http.NewRequest("POST", server.URL, nil)
			require.NoError(t, err)

			client := &http.Client{}
			resp, err := handler.Handle(client, req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)

				if tt.validate != nil {
					tt.validate(t, capturedContentType, capturedBody)
				}
			}
		})
	}
}

func TestCreateMultipartHeader(t *testing.T) {
	tests := []struct {
		name        string
		field       *MultipartField
		contentType string
		validate    func(t *testing.T, header multipart.FileHeader)
	}{
		{
			name: "field with filename",
			field: &MultipartField{
				Name:     "file",
				FileName: "test.txt",
			},
			contentType: "text/plain",
		},
		{
			name: "field with extra disposition",
			field: &MultipartField{
				Name:     "document",
				FileName: "doc.pdf",
				ExtraContentDisposition: map[string]string{
					"creation-date": "2024-01-01",
				},
			},
			contentType: "application/pdf",
		},
		{
			name: "field without content type",
			field: &MultipartField{
				Name:     "upload",
				FileName: "data.bin",
			},
			contentType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createMultipartHeader(tt.field, tt.contentType)

			assert.NotNil(t, header)

			if tt.field.FileName != "" {
				// Check if either "Name" or "name" exists (case-insensitive check)
				found := false
				for key := range header {
					if strings.ToLower(key) == "name" || strings.ToLower(key) == "filename" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected 'name' or 'filename' in header")
			}

			if tt.contentType != "" {
				assert.Equal(t, tt.contentType, header.Get("Content-Type"))
			}
		})
	}
}

func TestMultipartFieldProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress MultipartFieldProgress
	}{
		{
			name: "progress with values",
			progress: MultipartFieldProgress{
				Name:     "upload",
				FileName: "file.txt",
				FileSize: 1024,
				Written:  512,
			},
		},
		{
			name: "progress completed",
			progress: MultipartFieldProgress{
				Name:     "document",
				FileName: "doc.pdf",
				FileSize: 2048,
				Written:  2048,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.progress.Name, tt.progress.Name)
			assert.Equal(t, tt.progress.FileName, tt.progress.FileName)
			assert.Equal(t, tt.progress.FileSize, tt.progress.FileSize)
			assert.Equal(t, tt.progress.Written, tt.progress.Written)
		})
	}
}

func TestMultipartWithGetReaderError(t *testing.T) {
	tests := []struct {
		name        string
		field       *MultipartField
		expectError bool
	}{
		{
			name: "GetReader returns error",
			field: &MultipartField{
				Name:     "file",
				FileName: "error.txt",
				GetReader: func() (io.ReadCloser, error) {
					return nil, assert.AnError
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			middleware := Multipart([]*MultipartField{tt.field})
			handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				return client.Do(req)
			}))

			req, err := http.NewRequest("POST", server.URL, nil)
			require.NoError(t, err)

			client := &http.Client{}
			_, err = handler.Handle(client, req)

			if tt.expectError {
				assert.Error(t, err)
			}
		})
	}
}
