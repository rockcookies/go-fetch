package dump

import (
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Redactor
// ─────────────────────────────────────────────────────────────────────────────

// Redactor masks sensitive data before a DumpEntry is handed to the writer.
type Redactor interface {
	RedactHeaders(h http.Header) http.Header
	RedactBody(contentType string, body []byte) []byte
}

// NoopRedactor passes everything through unchanged.
type NoopRedactor struct{}

func (NoopRedactor) RedactHeaders(h http.Header) http.Header { return h }
func (NoopRedactor) RedactBody(_ string, b []byte) []byte    { return b }

// DefaultRedactor replaces the value of nominated header keys with "[REDACTED]".
// Keys are matched case-insensitively.
//
//	r := DefaultRedactor{
//	    Headers: map[string]struct{}{
//	        "authorization": {},
//	        "x-api-key":     {},
//	    },
//	}
type DefaultRedactor struct {
	Headers map[string]struct{}
}

func (r DefaultRedactor) RedactHeaders(h http.Header) http.Header {
	if len(h) == 0 || len(r.Headers) == 0 {
		return h
	}
	out := h.Clone()
	for k := range out {
		if _, ok := r.Headers[strings.ToLower(k)]; ok {
			out[k] = []string{"[REDACTED]"}
		}
	}
	return out
}

func (r DefaultRedactor) RedactBody(_ string, b []byte) []byte { return b }
