// Package dump provides HTTP request/response dumping and logging middleware.
package dump

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Options configures the dump middleware behavior including logging, filtering,
// and request/response body handling.
//
// Design philosophy:
//   - All fields are optional to support incremental adoption
//   - Filters compose to enable complex logging rules
//   - Body size limits protect against memory exhaustion
//   - Separate request/response controls allow asymmetric logging
//
// Why function-based configuration:
//   - Dynamic behavior based on request context (e.g., log full body for errors only)
//   - Avoids global state by passing request-specific decisions to callables
type Options struct {
	Skippers             []func(req *http.Request) bool
	Logger               *slog.Logger
	LogLevel             slog.Level
	LogLevelFunc         func(req *http.Request, status int) slog.Level
	Filters              []Filter
	ExtraAttrs           func(req *http.Request, status int) []slog.Attr
	RequestBodyFilter    func(req *http.Request) bool
	RequestBodyMaxSize   int64
	RequestHeaderFilter  func(key string, value []string) []any
	RequestAttrs         func(*http.Request) []any
	ResponseBodyFilter   func(req *http.Request) bool
	ResponseBodyMaxSize  int64
	ResponseHeaderFilter func(key string, value []string) []any
	ResponseAttrs        func(*http.Response, time.Duration) []any
}

// DefaultOptions returns sensible default options for the dump middleware.
// Sets 10KB max for request body and 100KB max for response body.
//
// Why these limits:
//   - 10KB request body: Typical for API payloads; prevents DoS via large POST bodies
//   - 100KB response body: Accommodates JSON responses; avoids logging binary files
//   - Error-aware log levels: 5xx=error, 4xx=warn, 2xx=info for proper alerting
//   - OPTIONS at debug: Preflight requests are noise in production logs
//
// These defaults balance observability with performance and security.
func DefaultOptions() *Options {
	return &Options{
		Logger:   slog.Default(),
		LogLevel: slog.LevelInfo,
		LogLevelFunc: func(req *http.Request, statusCode int) (lvl slog.Level) {
			switch {
			case statusCode >= 500:
				lvl = slog.LevelError
			case statusCode == 429:
				lvl = slog.LevelInfo
			case statusCode >= 400:
				lvl = slog.LevelWarn
			case req.Method == "OPTIONS":
				lvl = slog.LevelDebug
			default:
				lvl = slog.LevelInfo
			}
			return
		},
		RequestBodyMaxSize:  1024 * 10,  // 10KB
		ResponseBodyMaxSize: 1024 * 100, // 100KB
	}
}

var _ http.RoundTripper = (*RoundTripper)(nil)

// RoundTripper is an http.RoundTripper that logs request and response details.
type RoundTripper struct {
	next        http.RoundTripper
	optionsFunc func(req *http.Request) *Options
}

// NewRoundTripper creates a new dump RoundTripper with dynamic options.
// The optionsFunc is called for each request to determine logging behavior.
//
// Why dynamic options:
//   - Different endpoints may require different logging verbosity
//   - Request context (headers, auth) can determine if body should be logged
//   - Allows runtime adjustments without restarting the application
//
// Example use case:
//   - Log full bodies for /debug/* endpoints
//   - Omit sensitive headers for /auth/* endpoints
//   - Reduce logging for health checks
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

// RoundTrip implements http.RoundTripper by logging the request and response.
//
// Execution flow:
//  1. Evaluate options and skippers before any work (fast path)
//  2. Conditionally drain request body if configured (preserves stream)
//  3. Execute wrapped RoundTripper (the actual HTTP call)
//  4. Apply filters based on status code (defer ensures cleanup)
//  5. Conditionally drain response body if configured
//  6. Log structured data to slog
//
// Why defer for logging:
//   - Captures response status and errors even if RoundTrip panics
//   - Ensures duration measurement includes full request lifecycle
//   - Filters can skip logging based on response status (only known after RoundTrip)
func (rt *RoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	options := rt.optionsFunc(req)
	if options == nil {
		options = DefaultOptions()
	}

	for _, skip := range options.Skippers {
		if skip(req) {
			return rt.next.RoundTrip(req)
		}
	}

	dumpRequestBody := options.RequestBodyFilter != nil && options.RequestBodyFilter(req)
	dumpResponseBody := options.ResponseBodyFilter != nil && options.ResponseBodyFilter(req)

	var requestBody *drainedBody

	if dumpRequestBody {
		requestBody, req.Body, err = drainBody(req.Body, options.RequestBodyMaxSize)
		if err != nil {
			return nil, err
		}
	}

	start := time.Now()

	defer func() {
		duration := time.Since(start)
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		// Filtering
		for _, filter := range options.Filters {
			if !filter(req, statusCode) {
				return
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

		logger := options.Logger
		if logger == nil {
			logger = slog.Default()
		}

		level := options.LogLevel
		if options.LogLevelFunc != nil {
			level = options.LogLevelFunc(req, statusCode)
		}

		if !logger.Enabled(req.Context(), level) {
			return
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
			slog.Group("request_headers", getHeaderAttrs(req.Header, options.RequestHeaderFilter)...),
			slog.Group("request_body", getDrainedBodyAttrs(requestBody)...),
		}

		if resp != nil {
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
				slog.Group("response_headers", getHeaderAttrs(resp.Header, options.ResponseHeaderFilter)...),
				slog.Group("response_body", getDrainedBodyAttrs(responseBody)...),
			)
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		if options.ExtraAttrs != nil {
			extra := options.ExtraAttrs(req, statusCode)
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

// formatDuration formats a duration for log output with microsecond precision.
// Why microseconds: Balance between readability (ms too coarse) and verbosity (ns too detailed).
func formatDuration(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}

// scheme returns the URL scheme (http or https) based on request TLS state.
// Why check TLS: Request.URL.Scheme is often empty in RoundTripper context.
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// requestURL reconstructs the full URL from request components.
// Why reconstruct: Request.URL may be relative in client context; full URL aids debugging.
func requestURL(r *http.Request) string {
	return fmt.Sprintf("%s://%s%s", scheme(r), r.Host, r.URL)
}

// getDrainedBodyAttrs converts a drainedBody into structured log attributes.
// Why separate size and truncated: Operators need to know if logs show incomplete data.
// Returns empty slice for nil bodies to avoid cluttering logs with null values.
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

// getHeaderAttrs converts HTTP headers into structured log attributes.
// Why optional filter: Allows redacting sensitive headers (Authorization, Cookie) per-request.
// Single-value headers use string, multi-value use array for cleaner log output.
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
