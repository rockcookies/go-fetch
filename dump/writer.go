package dump

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// DumpWriter
// ─────────────────────────────────────────────────────────────────────────────

// DumpWriter is the output sink for dump entries.
// Implementations must be safe for concurrent use.
type DumpWriter interface {
	Write(ctx context.Context, entry DumpEntry) error
}

// ─────────────────────────────────────────────────────────────────────────────
// SlogWriter
// ─────────────────────────────────────────────────────────────────────────────

// SlogLevelFunc decides the slog level for a dump entry.
// m is the HTTP method, statusCode is the response status (0 when unavailable).
type SlogLevelFunc func(ctx context.Context, e DumpEntry) slog.Level

// DefaultLevelFunc is the default LevelFunc used by SlogWriter.
// Priority (highest first):
//  1. TransportError or PanicInfo → ERROR
//  2. status >= 500              → ERROR
//  3. status == 429              → INFO
//  4. status >= 400              → WARN
//  5. method == "OPTIONS"        → DEBUG
//  6. otherwise                  → INFO
func DefaultLevelFunc(ctx context.Context, e DumpEntry) slog.Level {
	switch {
	case e.Meta.TransportError != nil || e.Meta.PanicInfo != nil:
		return slog.LevelError
	case e.Meta.Status >= 500:
		return slog.LevelError
	case e.Meta.Status == 429:
		return slog.LevelInfo
	case e.Meta.Status >= 400:
		return slog.LevelWarn
	case e.Meta.Method == "OPTIONS":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// SlogWriter emits dump entries via log/slog.
// Level is determined by LevelFunc; if nil, DefaultLevelFunc is used.
type SlogWriter struct {
	Logger    *slog.Logger
	LevelFunc SlogLevelFunc
}

func (w *SlogWriter) Write(ctx context.Context, e DumpEntry) error {
	attrs := []any{
		slog.String("method", e.Meta.Method),
		slog.String("url", e.Meta.URL),
		slog.Int("status", e.Meta.Status),
		slog.Duration("latency", e.Meta.Latency),
	}
	if e.Meta.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", e.Meta.TraceID))
	}
	if e.Meta.ReqID != "" {
		attrs = append(attrs, slog.String("req_id", e.Meta.ReqID))
	}
	if e.Meta.TransportError != nil {
		attrs = append(attrs, slog.String("transport_error", e.Meta.TransportError.Error()))
	}
	if e.Meta.PanicInfo != nil {
		attrs = append(attrs,
			slog.String("panic", fmt.Sprintf("%v", e.Meta.PanicInfo.Value)),
			slog.String("stack", string(e.Meta.PanicInfo.Stack)),
		)
	}
	if len(e.ReqHeaders) > 0 {
		attrs = append(attrs, slog.Any("req_headers", map[string][]string(e.ReqHeaders)))
	}
	if len(e.RespHeaders) > 0 {
		attrs = append(attrs, slog.Any("resp_headers", map[string][]string(e.RespHeaders)))
	}
	if len(e.ReqBody) > 0 {
		attrs = append(attrs, slog.String("req_body", string(e.ReqBody)))
		if e.Meta.ReqBodyTruncated {
			attrs = append(attrs, slog.Bool("req_body_truncated", true))
		}
	}
	if len(e.RespBody) > 0 {
		attrs = append(attrs, slog.String("resp_body", string(e.RespBody)))
		if e.Meta.RespBodyTruncated {
			attrs = append(attrs, slog.Bool("resp_body_truncated", true))
		}
	}

	lvlFn := w.LevelFunc
	if lvlFn == nil {
		lvlFn = DefaultLevelFunc
	}
	w.Logger.Log(ctx, lvlFn(ctx, e), "http dump", attrs...)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MultiWriter
// ─────────────────────────────────────────────────────────────────────────────

// MultiWriter fans out to multiple DumpWriters; all errors are collected and joined.
type MultiWriter struct{ Writers []DumpWriter }

func (m *MultiWriter) Write(ctx context.Context, e DumpEntry) error {
	var msgs []string
	for _, w := range m.Writers {
		if err := w.Write(ctx, e); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return fmt.Errorf("dump writers: %s", strings.Join(msgs, "; "))
}

// ─────────────────────────────────────────────────────────────────────────────
// NoopWriter
// ─────────────────────────────────────────────────────────────────────────────

// NoopWriter discards all entries.  Useful in tests or to disable dumping at runtime.
type NoopWriter struct{}

func (NoopWriter) Write(_ context.Context, _ DumpEntry) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────
// AsyncWriter
// ─────────────────────────────────────────────────────────────────────────────

// AsyncWriter wraps a DumpWriter and offloads writes to a background goroutine.
//
// When the internal queue is full, new entries are silently dropped to avoid
// blocking the HTTP hot path.
//
// Call Close() during graceful shutdown to drain the queue and wait for all
// in-flight writes to complete.
type AsyncWriter struct {
	inner DumpWriter
	ch    chan asyncEntry
	wg    sync.WaitGroup
}

type asyncEntry struct {
	ctx   context.Context
	entry DumpEntry
}

// NewAsyncWriter creates an AsyncWriter backed by w with a queue of queueSize.
// If queueSize <= 0 it defaults to 1024.
func NewAsyncWriter(w DumpWriter, queueSize int) *AsyncWriter {
	if queueSize <= 0 {
		queueSize = 1024
	}
	aw := &AsyncWriter{
		inner: w,
		ch:    make(chan asyncEntry, queueSize),
	}
	aw.wg.Add(1)
	go func() {
		defer aw.wg.Done()
		for item := range aw.ch {
			_ = aw.inner.Write(item.ctx, item.entry)
		}
	}()
	return aw
}

// Write enqueues an entry for async delivery.  Returns immediately; never blocks.
// The provided context is detached from cancellation (values are preserved) so
// tracing IDs remain accessible after the request context is cancelled.
func (w *AsyncWriter) Write(ctx context.Context, e DumpEntry) error {
	// Detach cancellation so tracing spans / values remain valid when the
	// background goroutine eventually calls inner.Write.  Requires Go 1.21+.
	ctx = context.WithoutCancel(ctx)
	select {
	case w.ch <- asyncEntry{ctx: ctx, entry: e}:
	default:
		// Queue full — drop silently rather than block the caller.
	}
	return nil
}

// Close drains the queue and waits for all pending writes to finish.
func (w *AsyncWriter) Close() {
	close(w.ch)
	w.wg.Wait()
}
