package dump

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// captureRequestBody
// ─────────────────────────────────────────────────────────────────────────────

func reqWithBody(body, contentType string) *http.Request {
	req := &http.Request{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

func TestCaptureRequestBodyNormal(t *testing.T) {
	const payload = "hello world"
	req := reqWithBody(payload, "application/json")

	got, truncated := captureRequestBody(req, 64*1024, false)

	if string(got) != payload {
		t.Errorf("captured %q, want %q", got, payload)
	}
	if truncated {
		t.Error("expected not truncated")
	}
	// Body must be fully restored for downstream transport.
	restored, _ := io.ReadAll(req.Body)
	if string(restored) != payload {
		t.Errorf("restored body %q, want %q", restored, payload)
	}
}

func TestCaptureRequestBodyTruncated(t *testing.T) {
	payload := strings.Repeat("a", 20)
	req := reqWithBody(payload, "text/plain")

	got, truncated := captureRequestBody(req, 10, false)

	if len(got) != 10 {
		t.Errorf("captured %d bytes, want 10", len(got))
	}
	if !truncated {
		t.Error("expected truncated=true")
	}
	// Downstream body must still be complete.
	restored, _ := io.ReadAll(req.Body)
	if string(restored) != payload {
		t.Errorf("restored body %q, want full payload", restored)
	}
}

func TestCaptureRequestBodyUnlimited(t *testing.T) {
	// maxBytes < 0 → unlimited: whole body is captured, no truncation.
	payload := strings.Repeat("x", 200*1024) // 200 KiB > DefaultBodyMaxBytes
	req := reqWithBody(payload, "text/plain")

	got, truncated := captureRequestBody(req, -1, false)

	if len(got) != len(payload) {
		t.Errorf("captured %d bytes, want %d", len(got), len(payload))
	}
	if truncated {
		t.Error("expected truncated=false for unlimited capture")
	}
	restored, _ := io.ReadAll(req.Body)
	if len(restored) != len(payload) {
		t.Errorf("restored body length %d, want %d", len(restored), len(payload))
	}
}

func TestCaptureRequestBodySkipBinary(t *testing.T) {
	req := reqWithBody("binary\x00data", "image/png")

	got, truncated := captureRequestBody(req, 64*1024, true /* skipBinary */)

	if got != nil || truncated {
		t.Errorf("expected nil capture for binary body, got %q truncated=%v", got, truncated)
	}
}

func TestCaptureRequestBodyTextNotSkipped(t *testing.T) {
	const payload = `{"key":"value"}`
	req := reqWithBody(payload, "application/json")

	got, _ := captureRequestBody(req, 64*1024, true /* skipBinary */)

	if string(got) != payload {
		t.Errorf("text body was unexpectedly skipped: got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// captureReadCloser
// ─────────────────────────────────────────────────────────────────────────────

func TestCaptureReadCloserNormal(t *testing.T) {
	const payload = "response body"
	rc := io.NopCloser(strings.NewReader(payload))

	var (
		callbackBody      []byte
		callbackTruncated bool
		callbackCount     int
	)
	crc := newCaptureReadCloser(rc, 64*1024, false, "text/plain", func(body []byte, truncated bool) {
		callbackBody = body
		callbackTruncated = truncated
		callbackCount++
	})

	data, _ := io.ReadAll(crc)
	_ = crc.Close()

	if string(data) != payload {
		t.Errorf("read %q, want %q", data, payload)
	}
	if string(callbackBody) != payload {
		t.Errorf("callback body %q, want %q", callbackBody, payload)
	}
	if callbackTruncated {
		t.Error("expected not truncated")
	}
	if callbackCount != 1 {
		t.Errorf("callback called %d times, want 1", callbackCount)
	}
}

func TestCaptureReadCloserTruncated(t *testing.T) {
	payload := strings.Repeat("b", 20)
	rc := io.NopCloser(strings.NewReader(payload))

	var cbTruncated bool
	var cbLen int
	crc := newCaptureReadCloser(rc, 10, false, "text/plain", func(body []byte, truncated bool) {
		cbLen = len(body)
		cbTruncated = truncated
	})

	_, _ = io.ReadAll(crc)
	_ = crc.Close()

	if cbLen != 10 {
		t.Errorf("captured %d bytes, want 10", cbLen)
	}
	if !cbTruncated {
		t.Error("expected truncated=true")
	}
}

func TestCaptureReadCloserUnlimited(t *testing.T) {
	payload := strings.Repeat("u", 200*1024) // 200 KiB
	rc := io.NopCloser(strings.NewReader(payload))

	var cbLen int
	var cbTruncated bool
	crc := newCaptureReadCloser(rc, -1, false, "text/plain", func(body []byte, truncated bool) {
		cbLen = len(body)
		cbTruncated = truncated
	})

	_, _ = io.ReadAll(crc)
	_ = crc.Close()

	if cbLen != len(payload) {
		t.Errorf("captured %d bytes, want %d", cbLen, len(payload))
	}
	if cbTruncated {
		t.Error("expected truncated=false for unlimited capture")
	}
}

func TestCaptureReadCloserIdempotentClose(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	var count int
	crc := newCaptureReadCloser(rc, 64*1024, false, "text/plain", func([]byte, bool) {
		count++
	})
	_, _ = io.ReadAll(crc)
	_ = crc.Close()
	_ = crc.Close() // second close must not fire callback again
	_ = crc.Close()

	if count != 1 {
		t.Errorf("callback fired %d times, want exactly 1", count)
	}
}

func TestCaptureReadCloserSkipBinary(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("\x00\x01binary"))
	var cbBody []byte
	crc := newCaptureReadCloser(rc, 64*1024, true /* skipBinary */, "image/jpeg", func(body []byte, _ bool) {
		cbBody = body
	})
	_, _ = io.ReadAll(crc)
	_ = crc.Close()

	if len(cbBody) != 0 {
		t.Errorf("expected no capture for binary content-type, got %d bytes", len(cbBody))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// isTextContent
// ─────────────────────────────────────────────────────────────────────────────

func TestIsTextContent(t *testing.T) {
	cases := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/xml", true},
		{"application/x-www-form-urlencoded", true},
		{"text/plain", true},
		{"text/html; charset=utf-8", true},
		{"image/png", false},
		{"application/octet-stream", false},
		{"", false},
		{"  TEXT/PLAIN  ", true}, // leading/trailing spaces + upper-case
	}
	for _, c := range cases {
		if got := isTextContent(c.ct); got != c.want {
			t.Errorf("isTextContent(%q) = %v, want %v", c.ct, got, c.want)
		}
	}
}
