package dump

import (
	"net/http"
	"runtime/debug"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DumpTransport
// ─────────────────────────────────────────────────────────────────────────────

// DumpTransport wraps an http.RoundTripper and dumps selected HTTP exchanges.
//
// It is safe for concurrent use.  All fields must be set before the first call
// to RoundTrip; mutating them afterwards is a data race.
//
// Example:
//
//	client := &http.Client{
//	    Transport: &DumpTransport{
//	        Next:    http.DefaultTransport,
//	        Options: DumpOptions{Parts: DumpAll},
//	        Writer:  &SlogWriter{Logger: slog.Default()},
//	    },
//	}
type DumpTransport struct {
	// Next is the underlying RoundTripper.  Nil → http.DefaultTransport.
	Next http.RoundTripper

	// PreFilter is evaluated before the request is sent.
	// resp and err are always nil at this point.
	// Nil → always pass.
	PreFilter Filter

	// PostFilter is evaluated after the response arrives (or an error occurs).
	// Nil → always pass.
	PostFilter Filter

	Options DumpOptions

	// Writer receives the completed DumpEntry.  Nil → no-op.
	Writer DumpWriter

	// Redactor masks sensitive data.  Nil → NoopRedactor.
	Redactor Redactor

	// MetaExtractor pulls tracing IDs from the request context.  Nil → skipped.
	MetaExtractor MetaExtractor
}

var _ http.RoundTripper = (*DumpTransport)(nil)

func (t *DumpTransport) roundTripper() http.RoundTripper {
	if t.Next != nil {
		return t.Next
	}
	return http.DefaultTransport
}

func (t *DumpTransport) redactor() Redactor {
	if t.Redactor != nil {
		return t.Redactor
	}
	return NoopRedactor{}
}

func (t *DumpTransport) preFilter() Filter {
	if t.PreFilter != nil {
		return t.PreFilter
	}
	return AlwaysFilter
}

func (t *DumpTransport) postFilter() Filter {
	if t.PostFilter != nil {
		return t.PostFilter
	}
	return AlwaysFilter
}

// RoundTrip implements http.RoundTripper.
func (t *DumpTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	start := time.Now()

	// ── pre-filter: decide early without reading body ─────────────────────
	dumpThis := t.preFilter().Match(req, nil, nil)

	// ── capture request body (only when pre-filter passed) ────────────────
	var reqBody []byte
	var reqTruncated bool
	if dumpThis && req.Body != nil && req.Body != http.NoBody {
		reqBody, reqTruncated = captureRequestBody(req, bodyMaxBytes(t.Options), t.Options.SkipBinaryBody)
	}

	// ── call inner transport; recover panics in an isolated closure ────────
	//
	// The closure boundary is intentional: recover() only catches panics from
	// the same goroutine and only when called directly inside a deferred
	// function.  Placing RoundTrip inside the closure means the outer stack
	// frame is clean when we re-panic, and debug.Stack() captures only the
	// inner transport's frames.
	var panicInfo *PanicInfo
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicInfo = &PanicInfo{Value: r, Stack: debug.Stack()}
			}
		}()
		resp, err = t.roundTripper().RoundTrip(req)
	}()

	latency := time.Since(start)

	// ── panic path: dump then re-panic ────────────────────────────────────
	if panicInfo != nil {
		if dumpThis {
			t.flushEntry(dumpArgs{
				req:          req,
				reqBody:      reqBody,
				reqTruncated: reqTruncated,
				latency:      latency,
				panicInfo:    panicInfo,
			})
		}
		panic(panicInfo.Value) // re-panic with original value; stack is clean
	}

	// ── transport error path ──────────────────────────────────────────────
	if err != nil {
		if dumpThis && t.postFilter().Match(req, nil, err) {
			t.flushEntry(dumpArgs{
				req:          req,
				transportErr: err,
				reqBody:      reqBody,
				reqTruncated: reqTruncated,
				latency:      latency,
			})
		}
		// Preserve resp per http.RoundTripper contract: a non-nil resp alongside
		// a non-nil err can occur (e.g. after a redirect failure).
		return resp, err
	}

	// ── post-filter ───────────────────────────────────────────────────────
	if !dumpThis || !t.postFilter().Match(req, resp, nil) {
		return resp, nil
	}

	// ── streaming response body capture ───────────────────────────────────
	//
	// We tee-read rather than buffer upfront so the caller's Read calls
	// proceed normally.  flushEntry is triggered when the caller closes the
	// body, at which point we have the complete captured slice.
	if t.Options.Parts&DumpResponseBody != 0 && resp.Body != nil {
		resp.Body = newCaptureReadCloser(
			resp.Body,
			bodyMaxBytes(t.Options),
			t.Options.SkipBinaryBody,
			resp.Header.Get("Content-Type"),
			func(body []byte, bodyTruncated bool) {
				t.flushEntry(dumpArgs{
					req:           req,
					resp:          resp,
					reqBody:       reqBody,
					respBody:      body,
					reqTruncated:  reqTruncated,
					respTruncated: bodyTruncated,
					latency:       latency,
				})
			},
		)
		return resp, nil
	}

	// ── no response-body capture: write immediately ────────────────────────
	t.flushEntry(dumpArgs{
		req:          req,
		resp:         resp,
		reqBody:      reqBody,
		reqTruncated: reqTruncated,
		latency:      latency,
	})
	return resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// flushEntry (internal)
// ─────────────────────────────────────────────────────────────────────────────

// dumpArgs groups all parameters for a single dump flush, avoiding a long
// argument list and keeping call sites readable.
type dumpArgs struct {
	req           *http.Request
	resp          *http.Response
	transportErr  error
	reqBody       []byte
	respBody      []byte
	reqTruncated  bool
	respTruncated bool
	latency       time.Duration
	panicInfo     *PanicInfo
}

func (t *DumpTransport) flushEntry(a dumpArgs) {
	if t.Writer == nil {
		return
	}

	redact := t.redactor()

	entry := DumpEntry{
		Meta: DumpMeta{
			Method:            a.req.Method,
			URL:               a.req.URL.String(),
			Status:            responseStatus(a.resp),
			Latency:           a.latency,
			TransportError:    a.transportErr,
			ReqBodyTruncated:  a.reqTruncated,
			RespBodyTruncated: a.respTruncated,
			ReqContentType:    a.req.Header.Get("Content-Type"),
			PanicInfo:         a.panicInfo,
		},
	}
	if a.resp != nil {
		entry.Meta.RespContentType = a.resp.Header.Get("Content-Type")
	}
	if t.MetaExtractor != nil {
		entry.Meta.TraceID, entry.Meta.ReqID = t.MetaExtractor(a.req.Context())
	}
	if t.Options.Parts&DumpRequestHeaders != 0 {
		entry.ReqHeaders = redact.RedactHeaders(a.req.Header)
	}
	if t.Options.Parts&DumpResponseHeaders != 0 && a.resp != nil {
		entry.RespHeaders = redact.RedactHeaders(a.resp.Header)
	}
	if t.Options.Parts&DumpRequestBody != 0 {
		entry.ReqBody = redact.RedactBody(entry.Meta.ReqContentType, a.reqBody)
	}
	if t.Options.Parts&DumpResponseBody != 0 {
		entry.RespBody = redact.RedactBody(entry.Meta.RespContentType, a.respBody)
	}

	// Writer errors are intentionally swallowed to avoid masking transport errors.
	_ = t.Writer.Write(a.req.Context(), entry)
}

func responseStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
