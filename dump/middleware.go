// Package dump provides HTTP request/response dumping and logging middleware.
package dump

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var _ http.RoundTripper = (*RoundTripper)(nil)

// RoundTripper is an http.RoundTripper that logs request and response details.
type RoundTripper struct {
	next        http.RoundTripper
	optionsFunc func(req *http.Request) *Options
}

// NewRoundTripper creates a dump RoundTripper with per-request dynamic options.
// This enables runtime adjustments (e.g., verbose /debug/* logs, redacted /auth/* headers)
// without restarts or global state.
func NewRoundTripper(next http.RoundTripper, optionsFunc func(req *http.Request) *Options) *RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &RoundTripper{
		next:        next,
		optionsFunc: optionsFunc,
	}
}

// NewRoundTripperWithOptions creates a new dump RoundTripper with fixed options.
func NewRoundTripperWithOptions(next http.RoundTripper, options *Options) *RoundTripper {
	return NewRoundTripper(next, func(req *http.Request) *Options {
		return options
	})
}

// RoundTrip implements http.RoundTripper with structured logging.
// Logs via defer to capture response/errors even on panics and enable
// status-based filtering (e.g., skip 2xx in production).
func (rt *RoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	options := rt.optionsFunc(req)
	if options == nil {
		options = DefaultOptions()
	}

	for _, filter := range options.Filters {
		if !filter(req) {
			return rt.next.RoundTrip(req)
		}
	}

	dumpRequestBody := false
	if len(options.RequestBodyFilters) > 0 {
		dumpRequestBody = true

		for _, filter := range options.RequestBodyFilters {
			if !filter(req) {
				dumpRequestBody = false
				break
			}
		}
	}

	var requestBody *drainedBody
	if dumpRequestBody {
		requestBody, req.Body, err = drainBody(req.Body, options.RequestBodyMaxSize)
		if err != nil {
			return nil, fmt.Errorf("dump middleware: failed to drain request body: %w", err)
		}
	}

	start := time.Now()

	defer func() {
		duration := time.Since(start)

		// Filtering
		for _, filter := range options.LogFilters {
			if !filter(req, resp, err) {
				return
			}
		}

		level := options.LogLevel
		if options.LogLevelFunc != nil {
			level = options.LogLevelFunc(req, resp, err)
		}

		logger := options.Logger
		if logger == nil {
			logger = slog.Default()
		}

		if !logger.Enabled(req.Context(), level) {
			return
		}

		dumpResponseBody := false
		if len(options.RequestBodyFilters) > 0 {
			dumpResponseBody = true

			for _, filter := range options.ResponseBodyFilters {
				if !filter(req, resp, err) {
					dumpResponseBody = false
					break
				}
			}
		}

		var responseBody *drainedBody
		if resp != nil {
			if dumpResponseBody {
				responseBody, resp.Body, err = drainBody(resp.Body, options.ResponseBodyMaxSize)
				if err != nil {
					return
				}
			}
		}

		reqGroup := []any{
			slog.String("method", req.Method),
			slog.String("proto", fmt.Sprintf("HTTP/%d.%d", req.ProtoMajor, req.ProtoMinor)),
			slog.String("host", req.Host),
			slog.String("path", req.URL.Path),
			slog.String("query", req.URL.RawQuery),
		}

		if options.RequestAttrs != nil {
			reqGroup = append(reqGroup, options.RequestAttrs(req)...)
		}

		attrs := []slog.Attr{
			slog.String("duration_ms", formatDuration(duration)),
			slog.Group("request", reqGroup...),
			slog.Group("request_headers", getHeaderAttrs(req.Header, options.RequestHeaderFormatter)...),
			slog.Group("request_body", getDrainedBodyAttrs(requestBody)...),
		}

		statusCode := 0

		if resp != nil {
			statusCode = resp.StatusCode

			respGroup := []any{
				slog.Int("status", resp.StatusCode),
				slog.String("status_text", http.StatusText(resp.StatusCode)),
			}

			if resp.ContentLength >= 0 {
				respGroup = append(respGroup, slog.Int64("content_length", resp.ContentLength))
			}

			if options.ResponseAttrs != nil {
				respGroup = append(respGroup, options.ResponseAttrs(resp, duration)...)
			}

			attrs = append(attrs,
				slog.Group("response", respGroup...),
				slog.Group("response_headers", getHeaderAttrs(resp.Header, options.ResponseHeaderFormatter)...),
				slog.Group("response_body", getDrainedBodyAttrs(responseBody)...),
			)
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		if options.ExtraAttrs != nil {
			extra := options.ExtraAttrs(req, resp, err)
			attrs = append(attrs, extra...)
		}

		msg := fmt.Sprintf("%s %s => HTTP %v (%v)", req.Method, req.URL, statusCode, duration)

		logger.LogAttrs(
			req.Context(),
			level,
			msg,
			attrs...,
		)
	}()

	resp, err = rt.next.RoundTrip(req)
	return
}

// formatDuration formats duration with microsecond precision (balances readability vs detail).
func formatDuration(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}

// getDrainedBodyAttrs converts drainedBody to slog attributes.
// Separate size/truncated fields prevent misleading partial-content logs.
func getDrainedBodyAttrs(db *drainedBody) []any {
	if db == nil {
		return []any{}
	}

	attrs := []any{
		slog.String("content", db.body.String()),
		slog.Int64("size", db.size),
	}

	if db.truncated {
		attrs = append(attrs,
			slog.Bool("truncated", true),
			slog.Int64("captured_size", int64(db.body.Len())),
		)
	}

	return attrs
}

// getHeaderAttrs converts headers to slog attributes with optional per-request redaction.
// Single-value headers map to strings; multi-value to arrays.
func getHeaderAttrs(header http.Header, filter func(key string, value []string) []any) []any {
	attrs := make([]any, 0, len(header))
	for key := range header {
		vals := header.Values(key)

		if filter != nil {
			if res := filter(key, vals); len(res) > 0 {
				attrs = append(attrs, res...)
			}
		} else if len(vals) == 1 {
			attrs = append(attrs, slog.String(key, vals[0]))
		} else if len(vals) > 1 {
			attrs = append(attrs, slog.Any(key, vals))
		}
	}
	return attrs
}
