// Package dump provides HTTP request/response dumping and logging middleware.
package dump

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rockcookies/go-fetch/internal/utils"
)

// Options configures the dump middleware behavior including logging, filtering,
// and request/response body handling.
type Options struct {
	Skippers             []func(req *http.Request) bool
	Logger               *slog.Logger
	LogLevel             slog.Level
	LogLevelFunc         func(req *http.Request, status int) slog.Level
	Filters              []Filter
	RequestBodyFilter    func(req *http.Request) bool
	RequestBodyMaxSize   int64
	RequestHeaderFilter  func(key string, value []string) []any
	RequestAttrs         func(*http.Request) []slog.Attr
	ResponseBodyFilter   func(req *http.Request) bool
	ResponseBodyMaxSize  int64
	ResponseHeaderFilter func(key string, value []string) []any
	ResponseAttrs        func(*http.Response, time.Duration) []slog.Attr
}

// DefaultOptions returns sensible default options for the dump middleware.
// Sets 10KB max for request body and 100KB max for response body.
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

var skipKey = utils.NewContextKey[bool]("fetch_dump_skip")

// SkipDump returns a context that will skip dump logging for the request.
func SkipDump(ctx context.Context) context.Context {
	return skipKey.WithValue(ctx, true)
}

// NewRoundTripper creates a new dump RoundTripper with dynamic options.
// The optionsFunc is called for each request to determine logging behavior.
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
func (rt *RoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if skip, _ := skipKey.GetValue(req.Context()); skip {
		return rt.next.RoundTrip(req)
	}

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

		attrs := []slog.Attr{
			slog.Group("http",
				slog.String("method", req.Method),
				slog.String("url", requestURL(req)),
				slog.String("host", req.Host),
				slog.String("path", req.URL.Path),
				slog.String("query", req.URL.RawQuery),
				slog.String("proto", req.Proto),
			),
			slog.Duration("duration", duration),
			slog.String("duration_ms", formatDuration(duration)),
			slog.Group("request_headers", getHeaderAttrs(req.Header, options.RequestHeaderFilter)...),
			slog.Group("request_body", getDrainedBodyAttrs(requestBody)...),
		}

		if options.RequestAttrs != nil {
			attrs = append(attrs, options.RequestAttrs(req)...)
		}

		if resp != nil {
			respGroup := []any{
				slog.Int("status", resp.StatusCode),
				slog.String("status_text", http.StatusText(resp.StatusCode)),
			}

			if resp.ContentLength >= 0 {
				respGroup = append(respGroup, slog.Int64("content_length", resp.ContentLength))
			}

			attrs = append(attrs,
				slog.Group("response", respGroup...),
				slog.Group("response_headers", getHeaderAttrs(resp.Header, options.ResponseHeaderFilter)...),
				slog.Group("response_body", getDrainedBodyAttrs(responseBody)...),
			)

			if options.ResponseAttrs != nil {
				attrs = append(attrs, options.ResponseAttrs(resp, duration)...)
			}
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		logger.LogAttrs(
			req.Context(),
			level,
			buildLogMessage(resp, err),
			attrs...,
		)
	}()

	resp, err = rt.next.RoundTrip(req)
	return
}

func buildLogMessage(resp *http.Response, err error) string {
	if err != nil {
		return "HTTP request failed"
	}
	if resp == nil {
		return "HTTP request completed"
	}
	return "HTTP request completed"
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func requestURL(r *http.Request) string {
	return fmt.Sprintf("%s://%s%s", scheme(r), r.Host, r.URL)
}

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
