package dump

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request body capture
// ─────────────────────────────────────────────────────────────────────────────

// captureRequestBody reads the request body with a tee so the downstream
// transport still gets the full content.
//
// maxBytes must already be normalised by the caller (i.e. not zero; use
// bodyMaxBytes()).  A negative maxBytes captures the whole body without limit.
func captureRequestBody(req *http.Request, maxBytes int64, skipBinary bool) (captured []byte, truncated bool) {
	ct := req.Header.Get("Content-Type")
	if skipBinary && !isTextContent(ct) {
		return nil, false
	}

	// Tee into fullBuf so we can fully restore req.Body for the downstream
	// transport, regardless of how much we capture.
	var fullBuf bytes.Buffer
	tee := io.TeeReader(req.Body, &fullBuf)

	if maxBytes < 0 {
		// Unlimited: read everything into the capture buffer.
		capBytes, _ := io.ReadAll(tee)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(fullBuf.Bytes()))
		return capBytes, false
	}

	// Bounded: read at most maxBytes+1 to detect truncation without buffering
	// the full body twice.
	limited := io.LimitReader(tee, maxBytes+1)
	capBytes, _ := io.ReadAll(limited)

	// Drain the rest so fullBuf receives the complete original body.
	_, _ = io.Copy(io.Discard, tee)
	_ = req.Body.Close()

	req.Body = io.NopCloser(bytes.NewReader(fullBuf.Bytes()))

	if int64(len(capBytes)) > maxBytes {
		capBytes = capBytes[:maxBytes]
		truncated = true
	}
	return capBytes, truncated
}

// ─────────────────────────────────────────────────────────────────────────────
// Response body tee capture
// ─────────────────────────────────────────────────────────────────────────────

// captureReadCloser is an io.ReadCloser that tees data into an internal buffer
// up to maxBytes, then fires a callback on Close with the captured slice.
//
// Close is idempotent: the callback fires exactly once regardless of how many
// times Close is called (e.g. by http.Client redirect handling).
type captureReadCloser struct {
	rc        io.ReadCloser
	buf       bytes.Buffer
	maxBytes  int64
	truncated bool
	skip      bool
	done      func(body []byte, truncated bool)
	once      sync.Once
}

func newCaptureReadCloser(
	rc io.ReadCloser,
	maxBytes int64,
	skipBinary bool,
	contentType string,
	done func(body []byte, truncated bool),
) io.ReadCloser {
	return &captureReadCloser{
		rc:       rc,
		maxBytes: maxBytes,
		skip:     skipBinary && !isTextContent(contentType),
		done:     done,
	}
}

func (r *captureReadCloser) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if n > 0 && !r.skip {
		if r.maxBytes < 0 {
			// Unlimited: always write to the buffer.
			_, _ = r.buf.Write(p[:n])
		} else {
			remain := r.maxBytes - int64(r.buf.Len())
			if remain > 0 {
				write := int64(n)
				if write > remain {
					write = remain
					r.truncated = true
				}
				_, _ = r.buf.Write(p[:write])
			} else {
				r.truncated = true
			}
		}
	}
	return n, err
}

// Close is safe to call multiple times; the done callback fires exactly once.
func (r *captureReadCloser) Close() error {
	err := r.rc.Close()
	r.once.Do(func() {
		if r.done != nil {
			// Clone to decouple the entry from the buffer before handing off.
			r.done(cloneBytes(r.buf.Bytes()), r.truncated)
		}
	})
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

var textPrefixes = []string{
	"application/json",
	"application/xml",
	"application/x-www-form-urlencoded",
	"text/",
}

// isTextContent reports whether the Content-Type indicates human-readable text.
func isTextContent(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	for _, p := range textPrefixes {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// cloneBytes returns a copy of b, safe to retain after the source buffer is reused.
func cloneBytes(b []byte) []byte {
	return append([]byte(nil), b...)
}
