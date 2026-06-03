package dump

import (
	"context"
	"net/http"
	"time"
)

// Transport is an http.RoundTripper that dumps selected HTTP exchanges.
// Wrap an existing transport (or http.DefaultTransport) and assign it to
// http.Client.Transport.
//
//	client := &http.Client{
//	    Transport: dump.New(http.DefaultTransport, opts...),
//	}
type Transport struct {
	next             http.RoundTripper
	filter           Filter
	entryFilter      func(DumpEntry) bool
	options          DumpOptions
	writer           DumpWriter
	requestRedactor  Redactor
	responseRedactor Redactor
	extract          MetaExtractor
}

// Option configures a Transport.
type Option func(*Transport)

// WithFilter sets the filter that decides whether to dump each exchange.
// Default: dump everything.
func WithFilter(f Filter) Option { return func(t *Transport) { t.filter = f } }

// WithEntryFilter runs after capture, immediately before Write.
// Return false to skip that dump. nil writes all entries.
func WithEntryFilter(fn func(DumpEntry) bool) Option {
	return func(t *Transport) { t.entryFilter = fn }
}

// WithOptions sets the dump options.
func WithOptions(o DumpOptions) Option { return func(t *Transport) { t.options = o } }

// WithWriter sets the output sink.
func WithWriter(w DumpWriter) Option { return func(t *Transport) { t.writer = w } }

// WithRequestRedactor sets the redactor applied to captured request headers and body.
func WithRequestRedactor(r Redactor) Option {
	return func(t *Transport) { t.requestRedactor = r }
}

// WithResponseRedactor sets the redactor applied to captured response headers and body.
func WithResponseRedactor(r Redactor) Option {
	return func(t *Transport) { t.responseRedactor = r }
}

// WithMetaExtractor sets a function that extracts tracing IDs from the context.
func WithMetaExtractor(fn MetaExtractor) Option { return func(t *Transport) { t.extract = fn } }

// New wraps next with dump middleware. If next is nil, http.DefaultTransport is used.
func New(next http.RoundTripper, opts ...Option) *Transport {
	if next == nil {
		next = http.DefaultTransport
	}
	t := &Transport{
		next:   next,
		filter: alwaysFilter,
		writer: NoopWriter{},
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

func (t *Transport) requestRedact() Redactor {
	if t.requestRedactor != nil {
		return t.requestRedactor
	}
	return NoopRedactor{}
}

func (t *Transport) responseRedact() Redactor {
	if t.responseRedactor != nil {
		return t.responseRedactor
	}
	return NoopRedactor{}
}

func (t *Transport) writeEntry(ctx context.Context, e DumpEntry) {
	defer func() { recover() }()
	if t.entryFilter != nil && !t.entryFilter(e) {
		return
	}
	_ = t.writer.Write(ctx, e)
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	ctx := req.Context()

	var reqBody []byte
	var reqTruncated bool
	var reqBodySkipped bool
	if t.options.RequestBody && req.Body != nil && req.Body != http.NoBody {
		reqBody, reqTruncated, reqBodySkipped = captureRequestBody(
			req,
			t.options.maxBytes(),
			t.options.SkipBinaryBody,
			req.Header.Get("Content-Type"),
		)
	}

	resp, err := t.next.RoundTrip(req)

	if err != nil {
		if t.options.DumpOnError {
			entry := t.buildEntry(req, nil, reqBody, nil, reqTruncated, false, reqBodySkipped, time.Since(start), err)
			t.writeEntry(ctx, entry)
		}
		return nil, err
	}

	f := t.filter
	if f == nil {
		f = alwaysFilter
	}
	if !f.Match(req, resp) {
		return resp, nil
	}

	if resp.Body != nil && resp.Body != http.NoBody && t.options.ResponseBody {
		resp.Body = newTeeBody(
			resp.Body,
			t.options.maxBytes(),
			t.options.SkipBinaryBody,
			resp.Header.Get("Content-Type"),
			func(captured []byte, truncated bool) {
				entry := t.buildEntry(req, resp, reqBody, captured, reqTruncated, truncated, reqBodySkipped, time.Since(start), nil)
				t.writeEntry(ctx, entry)
			},
		)
		return resp, nil
	}

	entry := t.buildEntry(req, resp, reqBody, nil, reqTruncated, false, reqBodySkipped, time.Since(start), nil)
	t.writeEntry(ctx, entry)
	return resp, nil
}

func (t *Transport) buildEntry(
	req *http.Request,
	resp *http.Response,
	reqBody, respBody []byte,
	reqTruncated, respTruncated bool,
	reqBodySkipped bool,
	latency time.Duration,
	err error,
) DumpEntry {
	meta := DumpMeta{
		Method:            req.Method,
		URL:               req.URL.String(),
		Latency:           latency,
		ReqBodyTruncated:  reqTruncated,
		RespBodyTruncated: respTruncated,
		ReqBodySkipped:    reqBodySkipped,
		ReqContentType:    req.Header.Get("Content-Type"),
		Err:               err,
	}
	if resp != nil {
		meta.Status = resp.StatusCode
		meta.RespContentType = resp.Header.Get("Content-Type")
	}
	if t.extract != nil {
		meta.TraceID, meta.ReqID = t.extract(req.Context())
	}

	reqRedact := t.requestRedact()
	respRedact := t.responseRedact()

	entry := DumpEntry{Meta: meta}
	if t.options.RequestHeaders {
		entry.ReqHeaders = reqRedact.RedactHeaders(req.Header)
	}
	if t.options.RequestBody {
		entry.ReqBody = reqRedact.RedactBody(meta.ReqContentType, reqBody)
	}
	if t.options.ResponseHeaders && resp != nil {
		entry.RespHeaders = respRedact.RedactHeaders(resp.Header)
	}
	if t.options.ResponseBody {
		entry.RespBody = respRedact.RedactBody(meta.RespContentType, respBody)
	}
	return entry
}
