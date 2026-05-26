package dump

import (
	"net/http"
	"testing"
)

func TestNoopRedactorPassthrough(t *testing.T) {
	r := NoopRedactor{}

	h := http.Header{"Authorization": {"Bearer secret"}, "X-Api-Key": {"key123"}}
	if got := r.RedactHeaders(h); got == nil {
		t.Fatal("NoopRedactor should return the original header map")
	}
	if got := r.RedactHeaders(h)["Authorization"][0]; got != "Bearer secret" {
		t.Errorf("NoopRedactor changed Authorization: %q", got)
	}

	body := []byte(`{"password":"s3cr3t"}`)
	if string(r.RedactBody("application/json", body)) != string(body) {
		t.Error("NoopRedactor should not modify body")
	}
}

func TestDefaultRedactorHeaders(t *testing.T) {
	r := DefaultRedactor{
		Headers: map[string]struct{}{
			"authorization": {},
			"x-api-key":     {},
		},
	}

	h := http.Header{
		"Authorization": {"Bearer secret"},
		"X-Api-Key":     {"key123"},
		"Content-Type":  {"application/json"},
	}
	out := r.RedactHeaders(h)

	if out["Authorization"][0] != "[REDACTED]" {
		t.Errorf("Authorization not redacted: %q", out["Authorization"])
	}
	if out["X-Api-Key"][0] != "[REDACTED]" {
		t.Errorf("X-Api-Key not redacted: %q", out["X-Api-Key"])
	}
	if out["Content-Type"][0] != "application/json" {
		t.Errorf("Content-Type should be untouched, got %q", out["Content-Type"])
	}
	// Original must not be mutated.
	if h["Authorization"][0] != "Bearer secret" {
		t.Error("DefaultRedactor mutated the original header map")
	}
}

func TestDefaultRedactorEmptyHeaders(t *testing.T) {
	r := DefaultRedactor{
		Headers: map[string]struct{}{"authorization": {}},
	}
	// Empty input → returned as-is without cloning.
	out := r.RedactHeaders(http.Header{})
	if len(out) != 0 {
		t.Errorf("expected empty header, got %v", out)
	}
}

func TestDefaultRedactorNoSensitiveKeys(t *testing.T) {
	r := DefaultRedactor{Headers: map[string]struct{}{}} // empty blocklist
	h := http.Header{"Authorization": {"Bearer secret"}}
	out := r.RedactHeaders(h)
	// Nothing to redact → returned unchanged.
	if out["Authorization"][0] != "Bearer secret" {
		t.Errorf("unexpected change: %q", out["Authorization"])
	}
}

func TestDefaultRedactorBodyPassthrough(t *testing.T) {
	r := DefaultRedactor{Headers: map[string]struct{}{"authorization": {}}}
	body := []byte(`{"secret":"value"}`)
	if string(r.RedactBody("application/json", body)) != string(body) {
		t.Error("DefaultRedactor.RedactBody should pass body through unchanged")
	}
}
