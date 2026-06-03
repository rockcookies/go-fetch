package dump

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		want bool
	}{
		{name: "json", ct: "application/json", want: true},
		{name: "json_charset", ct: "application/json; charset=utf-8", want: true},
		{name: "problem_json", ct: "application/problem+json", want: true},
		{name: "vnd_api_json", ct: "application/vnd.api+json", want: true},
		{name: "xml", ct: "application/xml", want: true},
		{name: "form", ct: "application/x-www-form-urlencoded", want: true},
		{name: "text_plain", ct: "text/plain", want: true},
		{name: "octet_stream", ct: "application/octet-stream", want: false},
		{name: "image", ct: "image/png", want: false},
		{name: "empty", ct: "", want: false},
		{name: "whitespace", ct: "  TEXT/HTML  ", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isTextContent(tt.ct); got != tt.want {
				t.Errorf("isTextContent(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestCaptureRequestBodyGetBody(t *testing.T) {
	body := []byte("hello")
	req := httptestNewRequestWithGetBody(body)

	data, truncated, skipped := captureRequestBody(req, 1024, false, "text/plain")
	if skipped {
		t.Fatal("expected not skipped")
	}
	if truncated {
		t.Fatal("expected not truncated")
	}
	if string(data) != "hello" {
		t.Errorf("data = %q, want hello", data)
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("req.Body = %q, want hello", got)
	}
}

func TestCaptureRequestBodyStreamingSkipped(t *testing.T) {
	pr, pw := io.Pipe()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", pr)
	req.GetBody = nil

	_, _, skipped := captureRequestBody(req, 1024, false, "text/plain")
	if !skipped {
		t.Fatal("expected skipped for nil GetBody")
	}
	_ = pw.Close()
}

func TestCaptureRequestBodyTruncated(t *testing.T) {
	body := bytes.Repeat([]byte("a"), 10_000)
	req := httptestNewRequestWithGetBody(body)

	data, truncated, skipped := captureRequestBody(req, 1024, false, "text/plain")
	if skipped {
		t.Fatal("expected not skipped")
	}
	if !truncated {
		t.Fatal("expected truncated")
	}
	if len(data) != 1024 {
		t.Fatalf("len(data) = %d, want 1024", len(data))
	}
}

func httptestNewRequestWithGetBody(body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return req
}

func TestTeeBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		maxBytes  int64
		wantCap   string
		wantTrunc bool
	}{
		{name: "full_capture", body: "hello", maxBytes: 1024, wantCap: "hello"},
		{name: "truncated", body: "abcdefghij", maxBytes: 5, wantCap: "abcde", wantTrunc: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var captured []byte
			var truncated bool
			done := make(chan struct{}, 1)

			src := io.NopCloser(strings.NewReader(tt.body))
			tee := newTeeBody(src, tt.maxBytes, false, "text/plain", func(b []byte, tr bool) {
				captured = b
				truncated = tr
				done <- struct{}{}
			})

			out, err := io.ReadAll(tee)
			if err != nil {
				t.Fatal(err)
			}
			_ = tee.Close()
			<-done

			if string(out) != tt.body {
				t.Errorf("read = %q, want %q", out, tt.body)
			}
			if string(captured) != tt.wantCap {
				t.Errorf("captured = %q, want %q", captured, tt.wantCap)
			}
			if truncated != tt.wantTrunc {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTrunc)
			}
		})
	}
}

func TestTeeBodyFiresOnCloseOnly(t *testing.T) {
	var count int
	src := io.NopCloser(bytes.NewReader([]byte("abcdefghij")))
	tee := newTeeBody(src, 5, false, "text/plain", func([]byte, bool) {
		count++
	})
	_, _ = io.ReadAll(tee)
	if count != 0 {
		t.Fatalf("onDone called %d times before Close, want 0", count)
	}
	_ = tee.Close()
	_ = tee.Close()
	if count != 1 {
		t.Errorf("onDone called %d times, want 1", count)
	}
}
