package dump

import (
	"context"
	"net/http"
	"time"
)

// DumpOptions controls which parts of the exchange are captured.
// Metadata (method, URL, status, latency) is always populated regardless of
// these flags; the flags only control what is forwarded to the DumpWriter.
type DumpOptions struct {
	RequestHeaders  bool
	RequestBody     bool
	ResponseHeaders bool
	ResponseBody    bool

	// BodyMaxBytes caps how many bytes are read from each body for dumping.
	// 0 = use DefaultBodyMaxBytes. -1 = unlimited (use with care on large payloads).
	BodyMaxBytes int64

	// SkipBinaryBody suppresses capture when Content-Type is not text-like.
	// The body is still forwarded intact to the caller.
	SkipBinaryBody bool

	// DumpOnError forces a dump entry even when RoundTrip returns an error
	// (e.g. connection refused, timeout). resp will be nil in that case.
	DumpOnError bool
}

// DefaultBodyMaxBytes is the cap applied when BodyMaxBytes == 0.
const DefaultBodyMaxBytes = 64 * 1024 // 64 KiB

func (o *DumpOptions) maxBytes() int64 {
	if o.BodyMaxBytes == 0 {
		return DefaultBodyMaxBytes
	}
	return o.BodyMaxBytes
}

// DumpMeta carries per-exchange metadata passed to the DumpWriter.
type DumpMeta struct {
	// TraceID / ReqID are extracted from the context via MetaExtractor.
	TraceID string
	ReqID   string

	Method  string
	URL     string
	Status  int // 0 when resp is nil (error / panic)
	// Latency is time from RoundTrip start until the dump is written.
	// With ResponseBody enabled, that is after resp.Body is closed (includes body read).
	Latency time.Duration

	// Whether the body was capped at BodyMaxBytes.
	ReqBodyTruncated  bool
	RespBodyTruncated bool

	// ReqBodySkipped is true when RequestBody capture was requested but skipped
	// because req.GetBody is nil (streaming body, e.g. io.Pipe multipart upload).
	ReqBodySkipped bool

	ReqContentType  string
	RespContentType string

	// Err holds the RoundTrip error, if any. Non-nil only when DumpOnError is set.
	Err error
}

// MetaExtractor pulls trace / request IDs from the context.
// Implement to integrate with your tracing infrastructure.
type MetaExtractor func(ctx context.Context) (traceID, reqID string)

// DumpEntry is everything passed to a DumpWriter for one HTTP exchange.
type DumpEntry struct {
	Meta        DumpMeta
	ReqHeaders  http.Header
	ReqBody     []byte
	RespHeaders http.Header
	RespBody    []byte
}
