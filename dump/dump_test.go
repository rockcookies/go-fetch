package dump

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrainBody(t *testing.T) {
	tests := []struct {
		name          string
		body          io.ReadCloser
		maxSize       int64
		expectedSize  int64
		expectedData  string
		expectedTrunc bool
		expectError   bool
	}{
		{
			name:          "nil body",
			body:          nil,
			maxSize:       0,
			expectedSize:  0,
			expectedData:  "",
			expectedTrunc: false,
			expectError:   false,
		},
		{
			name:          "http.NoBody",
			body:          http.NoBody,
			maxSize:       0,
			expectedSize:  0,
			expectedData:  "",
			expectedTrunc: false,
			expectError:   false,
		},
		{
			name:          "normal body without limit",
			body:          io.NopCloser(strings.NewReader("hello world")),
			maxSize:       0,
			expectedSize:  11,
			expectedData:  "hello world",
			expectedTrunc: false,
			expectError:   false,
		},
		{
			name:          "normal body with sufficient limit",
			body:          io.NopCloser(strings.NewReader("hello world")),
			maxSize:       100,
			expectedSize:  11,
			expectedData:  "hello world",
			expectedTrunc: false,
			expectError:   false,
		},
		{
			name:          "body truncated by maxSize",
			body:          io.NopCloser(strings.NewReader("hello world")),
			maxSize:       5,
			expectedSize:  5,
			expectedData:  "hello",
			expectedTrunc: true,
			expectError:   false,
		},
		{
			name:          "empty body",
			body:          io.NopCloser(strings.NewReader("")),
			maxSize:       0,
			expectedSize:  0,
			expectedData:  "",
			expectedTrunc: false,
			expectError:   false,
		},
		{
			name:          "large body with limit",
			body:          io.NopCloser(strings.NewReader(strings.Repeat("a", 1000))),
			maxSize:       100,
			expectedSize:  100,
			expectedData:  strings.Repeat("a", 100),
			expectedTrunc: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, newBody, err := drainBody(tt.body, tt.maxSize)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.body == nil || tt.body == http.NoBody {
				assert.Nil(t, result)
				assert.Equal(t, http.NoBody, newBody)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedSize, result.size)
			assert.Equal(t, tt.expectedData, result.body.String())
			assert.Equal(t, tt.expectedTrunc, result.truncated)

			// Verify newBody can be read and contains the same data
			data, err := io.ReadAll(newBody)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedData, string(data))
		})
	}
}

func TestDrainBodyMultipleReads(t *testing.T) {
	// Test that drainBody preserves the body content for re-reading
	originalData := "test data for multiple reads"
	body := io.NopCloser(strings.NewReader(originalData))

	result, newBody, err := drainBody(body, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Read from the new body
	data1, err := io.ReadAll(newBody)
	require.NoError(t, err)
	assert.Equal(t, originalData, string(data1))

	// The drained body should still contain the original data
	assert.Equal(t, originalData, result.body.String())
}

func TestDrainBodyExactLimit(t *testing.T) {
	// Test edge case where body size exactly equals maxSize
	data := "exact"
	body := io.NopCloser(strings.NewReader(data))

	result, newBody, err := drainBody(body, int64(len(data)))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(len(data)), result.size)
	assert.Equal(t, data, result.body.String())
	assert.True(t, result.truncated)

	readData, err := io.ReadAll(newBody)
	require.NoError(t, err)
	assert.Equal(t, data, string(readData))
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}

func TestDrainBodyReadError(t *testing.T) {
	// Test error handling during read
	body := &errorReader{}

	result, returnedBody, err := drainBody(body, 0)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, body, returnedBody)
}

type closeErrorReader struct {
	*bytes.Reader
}

func (c *closeErrorReader) Close() error {
	return io.ErrClosedPipe
}

func TestDrainBodyCloseError(t *testing.T) {
	// Test error handling during close
	data := "test"
	body := &closeErrorReader{Reader: bytes.NewReader([]byte(data))}

	result, returnedBody, err := drainBody(body, 0)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, body, returnedBody)
}
