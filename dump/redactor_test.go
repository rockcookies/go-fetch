package dump

import (
	"net/http"
	"testing"
)

func TestRedactor(t *testing.T) {
	sensitive := http.Header{
		"Authorization": {"Bearer secret"},
		"X-Api-Key":     {"key123"},
		"Content-Type":  {"application/json"},
	}
	body := []byte(`{"password":"s3cr3t"}`)

	tests := []struct {
		name     string
		redactor Redactor
		check    func(t *testing.T, r Redactor)
	}{
		{
			name:     "noop_passthrough",
			redactor: NoopRedactor{},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				out := r.RedactHeaders(sensitive)
				if out["Authorization"][0] != "Bearer secret" {
					t.Errorf("Authorization changed: %q", out["Authorization"])
				}
				if string(r.RedactBody("application/json", body)) != string(body) {
					t.Error("body should be unchanged")
				}
			},
		},
		{
			name: "default_redacts_sensitive",
			redactor: DefaultRedactor{
				Headers: map[string]struct{}{
					"authorization": {},
					"x-api-key":     {},
				},
			},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				out := r.RedactHeaders(sensitive)
				if out["Authorization"][0] != "[REDACTED]" {
					t.Errorf("Authorization = %q, want [REDACTED]", out["Authorization"])
				}
				if out["X-Api-Key"][0] != "[REDACTED]" {
					t.Errorf("X-Api-Key = %q, want [REDACTED]", out["X-Api-Key"])
				}
				if out["Content-Type"][0] != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", out["Content-Type"])
				}
			},
		},
		{
			name: "default_no_mutate_original",
			redactor: DefaultRedactor{
				Headers: map[string]struct{}{"authorization": {}},
			},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				orig := http.Header{"Authorization": {"Bearer secret"}}
				_ = r.RedactHeaders(orig)
				if orig["Authorization"][0] != "Bearer secret" {
					t.Error("original header was mutated")
				}
			},
		},
		{
			name: "empty_headers",
			redactor: DefaultRedactor{
				Headers: map[string]struct{}{"authorization": {}},
			},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				out := r.RedactHeaders(http.Header{})
				if len(out) != 0 {
					t.Errorf("expected empty header, got %v", out)
				}
			},
		},
		{
			name:     "empty_blocklist",
			redactor: DefaultRedactor{Headers: map[string]struct{}{}},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				h := http.Header{"Authorization": {"Bearer secret"}}
				out := r.RedactHeaders(h)
				if out["Authorization"][0] != "Bearer secret" {
					t.Errorf("unexpected change: %q", out["Authorization"])
				}
			},
		},
		{
			name: "body_passthrough",
			redactor: DefaultRedactor{
				Headers: map[string]struct{}{"authorization": {}},
			},
			check: func(t *testing.T, r Redactor) {
				t.Helper()
				if string(r.RedactBody("application/json", body)) != string(body) {
					t.Error("DefaultRedactor.RedactBody should pass through")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, tt.redactor)
		})
	}
}
