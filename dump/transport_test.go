package dump

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────────────────────

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// okResp returns a minimal 200 response with an optional text body.
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

// buildTransport returns a DumpTransport wired to a captureWriter.
func buildTransport(opts DumpOptions, inner http.RoundTripper) (*DumpTransport, *captureWriter) {
	cw := &captureWriter{}
	dt := &DumpTransport{
		Next:    inner,
		Options: opts,
		Writer:  cw,
	}
	return dt, cw
}

// ─────────────────────────────────────────────────────────────────────────────
// Basic happy path
// ─────────────────────────────────────────────────────────────────────────────

func TestRoundTripDumpsEntry(t *testing.T) {
	dt, cw := buildTransport(DumpOptions{Parts: DumpAll}, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return okResp("hello"), nil
	}))

	req := httptest.NewRequest("GET", "/api/users", nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Drain the streaming response so the tee callback fires.
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	e := cw.entries[0]
	if e.Meta.Method != "GET" {
		t.Errorf("method = %q, want GET", e.Meta.Method)
	}
	if e.Meta.Status != 200 {
		t.Errorf("status = %d, want 200", e.Meta.Status)
	}
	if string(e.RespBody) != "hello" {
		t.Errorf("resp body = %q, want %q", e.RespBody, "hello")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Pre-filter skips dump
// ─────────────────────────────────────────────────────────────────────────────

func TestPreFilterSkip(t *testing.T) {
	cw := &captureWriter{}
	dt := &DumpTransport{
		Next:      roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		PreFilter: NotFilter(AlwaysFilter), // never matches
		Options:   DumpOptions{Parts: DumpAll},
		Writer:    cw,
	}

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := dt.RoundTrip(req)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 0 {
		t.Fatalf("pre-filter should have skipped dump, got %d entries", len(cw.entries))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Post-filter skips dump
// ─────────────────────────────────────────────────────────────────────────────

func TestPostFilterSkip(t *testing.T) {
	cw := &captureWriter{}
	dt := &DumpTransport{
		Next:       roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		PostFilter: StatusFilter([2]int{400, 599}), // only errors
		Options:    DumpOptions{Parts: DumpAll},
		Writer:     cw,
	}

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := dt.RoundTrip(req)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 0 {
		t.Fatalf("post-filter should have skipped 200 response, got %d entries", len(cw.entries))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Request body captured and restored
// ─────────────────────────────────────────────────────────────────────────────

func TestRequestBodyCapturedAndRestored(t *testing.T) {
	const payload = `{"name":"alice"}`
	var downstreamBody string

	inner := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		downstreamBody = string(b)
		return okResp(""), nil
	})

	dt, cw := buildTransport(DumpOptions{Parts: DumpRequestBody}, inner)
	req := httptest.NewRequest("POST", "/users",
		io.NopCloser(strings.NewReader(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(payload))

	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if downstreamBody != payload {
		t.Errorf("downstream body = %q, want %q", downstreamBody, payload)
	}
	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	if string(cw.entries[0].ReqBody) != payload {
		t.Errorf("captured req body = %q, want %q", cw.entries[0].ReqBody, payload)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Transport error path
// ─────────────────────────────────────────────────────────────────────────────

func TestTransportError(t *testing.T) {
	dialErr := errors.New("dial tcp: connection refused")
	inner := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, dialErr
	})

	dt, cw := buildTransport(DumpOptions{Parts: DumpAll}, inner)
	req := httptest.NewRequest("GET", "/", nil)

	_, err := dt.RoundTrip(req)
	if !errors.Is(err, dialErr) {
		t.Fatalf("expected dial error, got %v", err)
	}

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry on transport error, got %d", len(cw.entries))
	}
	e := cw.entries[0]
	if !errors.Is(e.Meta.TransportError, dialErr) {
		t.Errorf("TransportError = %v, want %v", e.Meta.TransportError, dialErr)
	}
	if e.Meta.Status != 0 {
		t.Errorf("status should be 0 on transport error, got %d", e.Meta.Status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Panic recovery
// ─────────────────────────────────────────────────────────────────────────────

func TestPanicRecovery(t *testing.T) {
	inner := roundTripFunc(func(*http.Request) (*http.Response, error) {
		panic("inner transport exploded")
	})

	dt, cw := buildTransport(DumpOptions{Parts: DumpAll}, inner)
	req := httptest.NewRequest("GET", "/", nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected a panic to propagate")
		}
		if len(cw.entries) != 1 {
			t.Errorf("expected 1 dump entry on panic, got %d", len(cw.entries))
		}
		if cw.entries[0].Meta.PanicInfo == nil {
			t.Error("PanicInfo should be set")
		}
	}()

	_, _ = dt.RoundTrip(req) //nolint:errcheck // intentionally panics
}

// ─────────────────────────────────────────────────────────────────────────────
// Redactor applied
// ─────────────────────────────────────────────────────────────────────────────

func TestRedactorApplied(t *testing.T) {
	cw := &captureWriter{}
	dt := &DumpTransport{
		Next:    roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		Options: DumpOptions{Parts: DumpRequestHeaders | DumpResponseHeaders},
		Writer:  cw,
		Redactor: DefaultRedactor{
			Headers: map[string]struct{}{"authorization": {}},
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, _ := dt.RoundTrip(req)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	got := cw.entries[0].ReqHeaders["Authorization"]
	if len(got) == 0 || got[0] != "[REDACTED]" {
		t.Errorf("Authorization header = %v, want [REDACTED]", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MetaExtractor
// ─────────────────────────────────────────────────────────────────────────────

func TestMetaExtractor(t *testing.T) {
	type traceKey struct{}
	ctx := context.WithValue(context.Background(), traceKey{}, "trace-xyz")

	cw := &captureWriter{}
	dt := &DumpTransport{
		Next:    roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp(""), nil }),
		Options: DumpOptions{Parts: DumpResponseHeaders},
		Writer:  cw,
		MetaExtractor: func(c context.Context) (string, string) {
			v, _ := c.Value(traceKey{}).(string)
			return v, "req-1"
		},
	}

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	resp, _ := dt.RoundTrip(req)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	e := cw.entries[0]
	if e.Meta.TraceID != "trace-xyz" {
		t.Errorf("TraceID = %q, want trace-xyz", e.Meta.TraceID)
	}
	if e.Meta.ReqID != "req-1" {
		t.Errorf("ReqID = %q, want req-1", e.Meta.ReqID)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NilWriter — no panic when Writer is nil
// ─────────────────────────────────────────────────────────────────────────────

func TestNilWriterNoPanic(t *testing.T) {
	dt := &DumpTransport{
		Next:    roundTripFunc(func(*http.Request) (*http.Response, error) { return okResp("hi"), nil }),
		Options: DumpOptions{Parts: DumpAll},
		Writer:  nil, // intentionally nil
	}

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// End-to-end with httptest.Server
// ─────────────────────────────────────────────────────────────────────────────

func TestEndToEndWithServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)

	cw := &captureWriter{}
	client := &http.Client{
		Transport: &DumpTransport{
			Next:    http.DefaultTransport,
			Options: DumpOptions{Parts: DumpAll},
			Writer:  cw,
		},
	}

	resp, err := client.Post(srv.URL+"/items", "application/json",
		strings.NewReader(`{"name":"test"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(cw.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cw.entries))
	}
	e := cw.entries[0]
	if e.Meta.Status != 201 {
		t.Errorf("status = %d, want 201", e.Meta.Status)
	}
	if string(e.RespBody) != `{"ok":true}` {
		t.Errorf("resp body = %q, want {\"ok\":true}", e.RespBody)
	}
	if string(e.ReqBody) != `{"name":"test"}` {
		t.Errorf("req body = %q", e.ReqBody)
	}
}
