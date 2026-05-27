package fetch

import (
	"bytes"
	"context"
	"errors"
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
			header, err := createMultipartHeader(tt.field, tt.contentType)
			require.NoError(t, err)
			assert.NotNil(t, header)

			if tt.field.FileName != "" {
				cd := header.Get("Content-Disposition")
				assert.Contains(t, cd, `name="`, "Expected name param in Content-Disposition")
				assert.Contains(t, cd, `filename="`, "Expected filename param in Content-Disposition")
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

func TestCreateMultipartHeaderInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "space in key", key: "invalid key"},
		{name: "semicolon in key", key: "key;injection"},
		{name: "equals in key", key: "key=value"},
		{name: "empty key", key: ""},
		{name: "control char in key", key: "key\x01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &MultipartField{
				Name:     "file",
				FileName: "test.txt",
				ExtraContentDisposition: map[string]string{
					tt.key: "value",
				},
			}
			_, err := createMultipartHeader(field, "text/plain")
			assert.Error(t, err)
		})
	}
}

func TestMultipartGoroutineNotLeakedOnHandlerError(t *testing.T) {
	// Verifies Issue #3: when the handler fails without reading the request body,
	// pr.CloseWithError unblocks the goroutine's pipe writes so it can exit cleanly.
	handlerErr := errors.New("handler error")
	field := &MultipartField{
		Name:        "file",
		FileName:    "test.txt",
		ContentType: "text/plain",
		FileSize:    100,
		GetReader: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(make([]byte, 100))), nil
		},
	}

	middleware := Multipart([]*MultipartField{field})
	handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		// Return error without reading req.Body; goroutine blocks on pipe write.
		return nil, handlerErr
	}))

	req, err := http.NewRequest("POST", "http://example.com", nil)
	require.NoError(t, err)

	_, err = handler.Handle(&http.Client{}, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, handlerErr)
}

func TestMultipartGoroutineErrorNotDropped(t *testing.T) {
	// Verifies Issue #1: goroutine errors are always received even when the
	// handler returns before the goroutine finishes (no select-default branch).
	readErr := errors.New("read error after first chunk")

	field := &MultipartField{
		Name:        "file",
		FileName:    "test.txt",
		ContentType: "text/plain",
		GetReader: func() (io.ReadCloser, error) {
			return &twoChunkReader{
				first: make([]byte, 512),
				err:   readErr,
			}, nil
		},
	}

	middleware := Multipart([]*MultipartField{field})
	handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		// Drain the body synchronously, mirroring what http.Client.Do guarantees:
		// the request body is fully consumed before returning.
		io.Copy(io.Discard, req.Body)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}))

	req, err := http.NewRequest("POST", "http://example.com", nil)
	require.NoError(t, err)

	_, err = handler.Handle(&http.Client{}, req)
	assert.ErrorIs(t, err, readErr)
}

// twoChunkReader returns first on the initial Read, then err on all subsequent reads.
type twoChunkReader struct {
	first    []byte
	err      error
	consumed bool
}

func (r *twoChunkReader) Read(p []byte) (int, error) {
	if !r.consumed {
		r.consumed = true
		n := copy(p, r.first)
		return n, nil
	}
	return 0, r.err
}

func (r *twoChunkReader) Close() error { return nil }

func TestCreateMultipartHeaderSortedKeys(t *testing.T) {
	field := &MultipartField{
		Name:     "file",
		FileName: "test.txt",
		ExtraContentDisposition: map[string]string{
			"z-param": "z-val",
			"a-param": "a-val",
			"m-param": "m-val",
		},
	}
	header, err := createMultipartHeader(field, "text/plain")
	require.NoError(t, err)

	cd := header.Get("Content-Disposition")
	aIdx := strings.Index(cd, "a-param")
	mIdx := strings.Index(cd, "m-param")
	zIdx := strings.Index(cd, "z-param")
	assert.Less(t, aIdx, mIdx, "a-param should appear before m-param")
	assert.Less(t, mIdx, zIdx, "m-param should appear before z-param")
}

func TestCreateMultipartHeaderNewlineRejected(t *testing.T) {
	tests := []struct {
		name  string
		field *MultipartField
	}{
		{
			name:  "newline in Name",
			field: &MultipartField{Name: "file\nname", FileName: "test.txt"},
		},
		{
			name:  "carriage return in Name",
			field: &MultipartField{Name: "file\rname", FileName: "test.txt"},
		},
		{
			name:  "newline in FileName",
			field: &MultipartField{Name: "file", FileName: "test\nfile.txt"},
		},
		{
			name:  "CRLF in FileName",
			field: &MultipartField{Name: "file", FileName: "test\r\nfile.txt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createMultipartHeader(tt.field, "text/plain")
			assert.Error(t, err)
		})
	}
}

func TestMultipartEmptyStreamNoAutoContentType(t *testing.T) {
	var partContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		part, err := mr.NextPart()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		partContentType = part.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	middleware := Multipart([]*MultipartField{
		{
			Name:     "empty",
			FileName: "empty.bin",
			GetReader: func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
	})
	handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		return client.Do(req)
	}))

	req, err := http.NewRequest("POST", server.URL, nil)
	require.NoError(t, err)

	resp, err := handler.Handle(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, partContentType, "empty stream should not have auto-detected Content-Type")
}

func TestMultipartContextCancellationPropagated(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel: ctx.Err() fires at first per-field check

	fields := []*MultipartField{
		{Name: "first", Values: []string{"a"}},
		{Name: "second", Values: []string{"b"}},
	}

	middleware := Multipart(fields)
	handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		go io.Copy(io.Discard, req.Body) // drain pipe so goroutine write does not block
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}))

	req, err := http.NewRequestWithContext(ctx, "POST", "http://example.com", nil)
	require.NoError(t, err)

	_, err = handler.Handle(&http.Client{}, req)
	assert.ErrorIs(t, err, context.Canceled)
}
