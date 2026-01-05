// Package dump provides HTTP request/response dumping and logging middleware.
package dump

import (
	"bytes"
	"io"
	"net/http"
)

// drainedBody holds the result of reading an HTTP body stream.
// It preserves both the raw content and metadata about the read operation.
//
// Why track truncation separately:
//   - Logging should indicate when content is incomplete
//   - Debugging requires knowing if the full payload was captured
//   - Size tracking enables proper Content-Length validation
type drainedBody struct {
	body      *bytes.Buffer // The captured content, ready for inspection
	size      int64         // Total bytes read (may exceed buffer if truncated)
	truncated bool          // Whether maxSize limit was hit during read
}

// drainBody reads and captures the body content while preserving it for subsequent use.
// This design allows middleware to inspect request/response bodies for logging without
// consuming the original stream.
//
// Why this approach:
//   - HTTP bodies are single-use streams; once read, they cannot be read again
//   - Middleware needs to log body content without breaking the request pipeline
//   - Memory is controlled via maxSize to prevent OOM on large payloads
//
// Parameters:
//   - b: The original body stream to read from
//   - maxSize: Maximum bytes to read (0 = unlimited). Prevents memory exhaustion.
//
// Returns:
//   - result: Captured body data with size and truncation info (nil for empty bodies)
//   - newBody: A fresh ReadCloser containing the same data for the next handler
//   - err: Read or close errors are propagated; caller must handle body cleanup
//
// Error handling:
//   - Read errors return the original body unchanged for fallback handling
//   - Close errors are returned as the body is already consumed
func drainBody(b io.ReadCloser, maxSize int64) (result *drainedBody, newBody io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return nil, http.NoBody, nil
	}

	var buf bytes.Buffer
	var reader io.Reader = b
	var totalRead int64
	var truncated bool

	if maxSize > 0 {
		reader = io.LimitReader(b, maxSize)
	}

	n, err := buf.ReadFrom(reader)
	totalRead = n
	if err != nil {
		return nil, b, err
	}

	if maxSize > 0 && n >= maxSize {
		truncated = true
	}

	if err = b.Close(); err != nil {
		return nil, b, err
	}

	return &drainedBody{
		body:      &buf,
		size:      totalRead,
		truncated: truncated,
	}, io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
