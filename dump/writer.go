package dump

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// DumpWriter is the output sink for captured exchanges.
type DumpWriter interface {
	Write(ctx context.Context, entry DumpEntry) error
}

// slogBodyPreviewMax caps req_body/resp_body string attributes in SlogWriter.
const slogBodyPreviewMax = 2048

func slogBodyPreview(b []byte) (string, bool) {
	if len(b) <= slogBodyPreviewMax {
		return string(b), false
	}
	return string(b[:slogBodyPreviewMax]), true
}

// SlogWriter emits dump entries via log/slog.
type SlogWriter struct {
	Logger *slog.Logger
	Level  slog.Level
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
	if e.Meta.Err != nil {
		attrs = append(attrs, slog.String("error", e.Meta.Err.Error()))
	}
	if len(e.ReqHeaders) > 0 {
		attrs = append(attrs, slog.Any("req_headers", map[string][]string(e.ReqHeaders)))
	}
	if len(e.ReqBody) > 0 {
		preview, previewTrunc := slogBodyPreview(e.ReqBody)
		attrs = append(attrs, slog.String("req_body", preview))
		if e.Meta.ReqBodyTruncated {
			attrs = append(attrs, slog.Bool("req_body_truncated", true))
		}
		if previewTrunc {
			attrs = append(attrs, slog.Bool("req_body_preview_truncated", true))
		}
	}
	if len(e.RespHeaders) > 0 {
		attrs = append(attrs, slog.Any("resp_headers", map[string][]string(e.RespHeaders)))
	}
	if len(e.RespBody) > 0 {
		preview, previewTrunc := slogBodyPreview(e.RespBody)
		attrs = append(attrs, slog.String("resp_body", preview))
		if e.Meta.RespBodyTruncated {
			attrs = append(attrs, slog.Bool("resp_body_truncated", true))
		}
		if previewTrunc {
			attrs = append(attrs, slog.Bool("resp_body_preview_truncated", true))
		}
	}
	w.Logger.Log(ctx, w.Level, "http dump", attrs...)
	return nil
}

// IOWriter writes a human-readable dump to any io.Writer.
type IOWriter struct{ W io.Writer }

func (w *IOWriter) Write(_ context.Context, e DumpEntry) error {
	var sb strings.Builder
	sb.WriteString("\n──── HTTP DUMP ────\n")
	sb.WriteString("→ " + e.Meta.Method + " " + e.Meta.URL + "\n")
	if e.Meta.TraceID != "" {
		sb.WriteString("   trace_id: " + e.Meta.TraceID + "\n")
	}
	if e.Meta.Err != nil {
		sb.WriteString("   error: " + e.Meta.Err.Error() + "\n")
	}
	if len(e.ReqHeaders) > 0 {
		sb.WriteString("   req headers:\n")
		for k, vs := range e.ReqHeaders {
			sb.WriteString("      " + k + ": " + strings.Join(vs, ", ") + "\n")
		}
	}
	if len(e.ReqBody) > 0 {
		sb.WriteString("   req body: " + string(e.ReqBody))
		if e.Meta.ReqBodyTruncated {
			sb.WriteString(" [truncated]")
		}
		sb.WriteString("\n")
	}
	if e.Meta.Status != 0 {
		sb.WriteString("← " + http.StatusText(e.Meta.Status) + "\n")
	}
	if len(e.RespHeaders) > 0 {
		sb.WriteString("   resp headers:\n")
		for k, vs := range e.RespHeaders {
			sb.WriteString("      " + k + ": " + strings.Join(vs, ", ") + "\n")
		}
	}
	if len(e.RespBody) > 0 {
		sb.WriteString("   resp body: " + string(e.RespBody))
		if e.Meta.RespBodyTruncated {
			sb.WriteString(" [truncated]")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("   latency: " + e.Meta.Latency.String() + "\n")
	sb.WriteString("──────────────────\n")
	_, err := io.WriteString(w.W, sb.String())
	return err
}

// MultiWriter fans out to multiple DumpWriters; all errors are joined.
type MultiWriter struct{ Writers []DumpWriter }

func (m *MultiWriter) Write(ctx context.Context, e DumpEntry) error {
	var errs []error
	for _, w := range m.Writers {
		if err := w.Write(ctx, e); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// NoopWriter discards everything (useful in tests or when toggling off at runtime).
type NoopWriter struct{}

func (NoopWriter) Write(_ context.Context, _ DumpEntry) error { return nil }
