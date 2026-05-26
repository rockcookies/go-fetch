package dump

import (
	"context"
	"net/http"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DumpPart bitmask
// ─────────────────────────────────────────────────────────────────────────────

// DumpPart is a bitmask that selects which parts of an exchange to capture.
type DumpPart uint8

const (
	DumpRequestHeaders  DumpPart = 1 << iota // capture request headers
	DumpRequestBody                          // capture request body
	DumpResponseHeaders                      // capture response headers
	DumpResponseBody                         // capture response body (streaming)

	// DumpAll captures everything.
	DumpAll DumpPart = DumpRequestHeaders | DumpRequestBody | DumpResponseHeaders | DumpResponseBody
)

// ─────────────────────────────────────────────────────────────────────────────
// DumpOptions
// ─────────────────────────────────────────────────────────────────────────────

// DumpOptions controls which parts of the exchange are captured.
type DumpOptions struct {
	// Parts is a bitmask of DumpPart values.
	Parts DumpPart

	// BodyMaxBytes caps bytes captured from each body (request or response).
	// 0 → DefaultBodyMaxBytes.  Use a negative value for unlimited.
	BodyMaxBytes int64

	// SkipBinaryBody skips body capture when Content-Type is not text-like.
	SkipBinaryBody bool
}

const DefaultBodyMaxBytes = 64 * 1024 // 64 KiB

// bodyMaxBytes returns the effective body cap from options.
// 0 → DefaultBodyMaxBytes; negative → unlimited (-1).
func bodyMaxBytes(o DumpOptions) int64 {
	if o.BodyMaxBytes == 0 {
		return DefaultBodyMaxBytes
	}
	return o.BodyMaxBytes
}

// ─────────────────────────────────────────────────────────────────────────────
// Metadata types
// ─────────────────────────────────────────────────────────────────────────────

// PanicInfo captures the value and stack trace of a recovered panic.
type PanicInfo struct {
	Value any
	Stack []byte // output of runtime/debug.Stack()
}

// DumpMeta carries per-exchange metadata included in every DumpEntry.
type DumpMeta struct {
	Method  string
	URL     string
	Status  int           // 0 when no response (transport error or panic)
	Latency time.Duration // time from first byte sent to response headers received

	// Populated by MetaExtractor when set.
	TraceID string
	ReqID   string

	ReqContentType  string
	RespContentType string

	ReqBodyTruncated  bool
	RespBodyTruncated bool

	// TransportError is set when the inner RoundTripper returns a non-nil error.
	TransportError error

	// PanicInfo is set when the inner RoundTripper panics.
	PanicInfo *PanicInfo
}

// MetaExtractor pulls tracing IDs out of the request context.
// Implement this to integrate with your observability stack.
type MetaExtractor func(ctx context.Context) (traceID, reqID string)

// ─────────────────────────────────────────────────────────────────────────────
// DumpEntry
// ─────────────────────────────────────────────────────────────────────────────

// DumpEntry is the complete record handed to a DumpWriter for one HTTP exchange.
type DumpEntry struct {
	Meta DumpMeta

	ReqHeaders  http.Header
	RespHeaders http.Header

	ReqBody  []byte
	RespBody []byte
}
