package dump

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type captureWriter struct {
	entries []DumpEntry
}

func (c *captureWriter) Write(_ context.Context, e DumpEntry) error {
	c.entries = append(c.entries, e)
	return nil
}

func okResp(body string) *http.Response {
	var rc io.ReadCloser = http.NoBody
	if body != "" {
		rc = io.NopCloser(strings.NewReader(body))
	}
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": {"text/plain"}},
		Body:       rc,
	}
}

func buildTransport(inner http.RoundTripper, cw *captureWriter, opts ...Option) *Transport {
	base := []Option{
		WithWriter(cw),
		WithOptions(DumpOptions{
			RequestHeaders:  true,
			RequestBody:     true,
			ResponseHeaders: true,
			ResponseBody:    true,
		}),
	}
	base = append(base, opts...)
	return New(inner, base...)
}

func TestRoundTripDumpsEntry(t *testing.T) {
	cw := &captureWriter{}
	dt := buildTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return okResp("hello"), nil
	}), cw)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/users", nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	e := cw.entries[0]
	if e.Meta.Method != http.MethodGet {
		t.Errorf("method = %q, want GET", e.Meta.Method)
	}
	if e.Meta.Status != 200 {
		t.Errorf("status = %d, want 200", e.Meta.Status)
	}
	if string(e.RespBody) != "hello" {
		t.Errorf("resp body = %q, want hello", e.RespBody)
	}
}

func TestFilterSkipsDump(t *testing.T) {
	cw := &captureWriter{}
	dt := buildTransport(
		roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		cw,
		WithFilter(StatusFilter([2]int{500, 599})),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp, _ := dt.RoundTrip(req)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 0 {
		t.Fatalf("filter should skip dump, got %d entries", len(cw.entries))
	}
}

func TestDumpOnError(t *testing.T) {
	cw := &captureWriter{}
	wantErr := errors.New("connection refused")
	dt := buildTransport(
		roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, wantErr
		}),
		cw,
		WithOptions(DumpOptions{
			RequestHeaders: true,
			DumpOnError:    true,
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	_, err := dt.RoundTrip(req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry on error, got %d", len(cw.entries))
	}
	if !errors.Is(cw.entries[0].Meta.Err, wantErr) {
		t.Errorf("Meta.Err = %v", cw.entries[0].Meta.Err)
	}
}

func TestTransportRedactorSides(t *testing.T) {
	reqRedact := DefaultRedactor{Headers: map[string]struct{}{"authorization": {}}}
	respRedact := DefaultRedactor{Headers: map[string]struct{}{"set-cookie": {}}}

	tests := []struct {
		name           string
		opts           []Option
		reqHeader      http.Header
		respHeader     http.Header
		checkReqAuth   string
		checkRespCookie string
		checkReqRaw    string
	}{
		{
			name: "request_redactor_only",
			opts: []Option{WithRequestRedactor(reqRedact)},
			reqHeader: http.Header{
				"Authorization": {"Bearer secret"},
				"Content-Type":  {"text/plain"},
			},
			respHeader: http.Header{
				"Set-Cookie": {"session=abc"},
			},
			checkReqAuth:    "[REDACTED]",
			checkRespCookie: "session=abc",
			checkReqRaw:     "Bearer secret",
		},
		{
			name: "response_redactor_only",
			opts: []Option{WithResponseRedactor(respRedact)},
			reqHeader: http.Header{
				"Authorization": {"Bearer secret"},
				"Content-Type":  {"text/plain"},
			},
			respHeader: http.Header{
				"Set-Cookie":   {"session=abc"},
				"Content-Type": {"text/plain"},
			},
			checkReqAuth:    "Bearer secret",
			checkRespCookie: "[REDACTED]",
			checkReqRaw:     "Bearer secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cw := &captureWriter{}
			dt := buildTransport(
				roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Header:     tt.respHeader.Clone(),
						Body:       http.NoBody,
					}, nil
				}),
				cw,
				append([]Option{WithOptions(DumpOptions{RequestHeaders: true, ResponseHeaders: true})}, tt.opts...)...,
			)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			req.Header = tt.reqHeader.Clone()
			resp, err := dt.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()

			if len(cw.entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(cw.entries))
			}
			e := cw.entries[0]

			if got := e.ReqHeaders.Get("Authorization"); got != tt.checkReqAuth {
				t.Errorf("dumped Authorization = %q, want %q", got, tt.checkReqAuth)
			}
			if got := e.RespHeaders.Get("Set-Cookie"); got != tt.checkRespCookie {
				t.Errorf("dumped Set-Cookie = %q, want %q", got, tt.checkRespCookie)
			}
			if got := req.Header.Get("Authorization"); got != tt.checkReqRaw {
				t.Errorf("live Authorization = %q, want %q (must not mutate req)", got, tt.checkReqRaw)
			}
		})
	}
}

func TestPipeRequestNoDeadlock(t *testing.T) {
	cw := &captureWriter{}
	dt := buildTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		_, _ = io.ReadAll(r.Body)
		return okResp("ok"), nil
	}), cw)

	pr, pw := io.Pipe()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/upload", pr)
	req.GetBody = nil
	go func() {
		_, _ = pw.Write([]byte("chunk"))
		_ = pw.Close()
	}()

	done := make(chan error, 1)
	go func() { _, err := dt.RoundTrip(req); done <- err }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RoundTrip: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTrip blocked (pipe deadlock)")
	}
}

func TestCaptureRequestBodyViaGetBody(t *testing.T) {
	cw := &captureWriter{}
	body := []byte(`{"id":1}`)
	dt := buildTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if string(got) != string(body) {
			t.Errorf("upstream body = %q", got)
		}
		return okResp(""), nil
	}), cw)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(cw.entries))
	}
	if string(cw.entries[0].ReqBody) != string(body) {
		t.Errorf("ReqBody = %q", cw.entries[0].ReqBody)
	}
}

func TestReqBodySkippedMeta(t *testing.T) {
	cw := &captureWriter{}
	dt := buildTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return okResp(""), nil
	}), cw)

	pr, pw := io.Pipe()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", pr)
	req.GetBody = nil
	go func() { _ = pw.Close() }()

	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(cw.entries))
	}
	if !cw.entries[0].Meta.ReqBodySkipped {
		t.Error("expected ReqBodySkipped")
	}
}

func TestEntryFilterSkipsWrite(t *testing.T) {
	cw := &captureWriter{}
	dt := buildTransport(
		roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp("secret"), nil }),
		cw,
		WithEntryFilter(func(e DumpEntry) bool {
			return !bytes.Contains(e.RespBody, []byte("secret"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 0 {
		t.Fatalf("entry filter should skip, got %d entries", len(cw.entries))
	}
}

type panicWriter struct{}

func (panicWriter) Write(context.Context, DumpEntry) error { panic("writer panic") }

func TestWriterPanicDoesNotFailRoundTrip(t *testing.T) {
	dt := buildTransport(
		roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		&captureWriter{},
		WithWriter(panicWriter{}),
		WithOptions(DumpOptions{ResponseHeaders: true}),
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip should succeed: %v", err)
	}
	_ = resp.Body.Close()
}
