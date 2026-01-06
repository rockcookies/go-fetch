// Package dump provides HTTP request/response dumping and logging middleware.
package dump

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	// DefaultRequestBodyMaxSize is the default maximum size for capturing request bodies (10KB).
	// This balances observability with DoS protection.
	DefaultRequestBodyMaxSize = 1024 * 10 // 10KB

	// DefaultResponseBodyMaxSize is the default maximum size for capturing response bodies (100KB).
	// This balances observability with log noise reduction.
	DefaultResponseBodyMaxSize = 1024 * 100 // 100KB
)

// Options configures dump middleware behavior.
// All fields are optional; filters compose for granular control.
// Function-based configs enable dynamic decisions per-request (e.g., log bodies only on errors).
type Options struct {
	Filters                 []func(req *http.Request) bool
	Logger                  *slog.Logger
	LogFilters              []func(req *http.Request, resp *http.Response, err error) bool
	LogLevel                slog.Level
	LogLevelFunc            func(req *http.Request, resp *http.Response, err error) slog.Level
	ExtraAttrs              func(req *http.Request, resp *http.Response, err error) []slog.Attr
	RequestBodyFilters      []func(req *http.Request) bool
	RequestBodyMaxSize      int64
	RequestHeaderFormatter  func(key string, value []string) []any
	RequestAttrs            func(*http.Request) []any
	ResponseBodyFilters     []func(req *http.Request, resp *http.Response, err error) bool
	ResponseBodyMaxSize     int64
	ResponseHeaderFormatter func(key string, value []string) []any
	ResponseAttrs           func(*http.Response, time.Duration) []any
}

// DefaultLogLevelFunc returns the appropriate log level based on response status and request method.
// 5xx errors log at ERROR level, 4xx at WARN (except 429 at INFO), OPTIONS at DEBUG, others at INFO.
func DefaultLogLevelFunc(req *http.Request, resp *http.Response, err error) (lvl slog.Level) {
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

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
}

// DefaultOptions returns defaults: 10KB request / 100KB response body limits,
// error-aware log levels (5xx=error, 4xx=warn, 2xx=info).
// These values balance observability with DoS protection and log noise reduction.
func DefaultOptions() *Options {
	return &Options{
		Logger:              slog.Default(),
		LogLevel:            slog.LevelInfo,
		LogLevelFunc:        DefaultLogLevelFunc,
		RequestBodyMaxSize:  DefaultRequestBodyMaxSize,
		ResponseBodyMaxSize: DefaultResponseBodyMaxSize,
	}
}
