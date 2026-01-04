package dump

import (
	"bytes"
	"io"
	"net/http"
)

type drainedBody struct {
	body      *bytes.Buffer
	size      int64
	truncated bool
}

// drainBody reads the entire body from an io.ReadCloser and returns both
// a drainedBody containing the read data and a new io.ReadCloser that can be used
// to re-read the same data. If maxSize > 0, only up to maxSize bytes are read.
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
