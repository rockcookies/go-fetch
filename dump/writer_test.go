package dump

import (
	"context"
	"errors"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// NoopWriter
// ─────────────────────────────────────────────────────────────────────────────

func TestNoopWriter(t *testing.T) {
	var w NoopWriter
	if err := w.Write(context.Background(), DumpEntry{}); err != nil {
		t.Errorf("NoopWriter.Write returned unexpected error: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MultiWriter
// ─────────────────────────────────────────────────────────────────────────────

type captureWriter struct {
	entries []DumpEntry
	err     error
}

func (w *captureWriter) Write(_ context.Context, e DumpEntry) error {
	w.entries = append(w.entries, e)
	return w.err
}

func TestMultiWriterAllOK(t *testing.T) {
	a, b := &captureWriter{}, &captureWriter{}
	m := &MultiWriter{Writers: []DumpWriter{a, b}}

	e := DumpEntry{Meta: DumpMeta{Method: "GET", URL: "/test", Status: 200}}
	if err := m.Write(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.entries) != 1 || len(b.entries) != 1 {
		t.Errorf("expected 1 entry each, got a=%d b=%d", len(a.entries), len(b.entries))
	}
}

func TestMultiWriterPartialError(t *testing.T) {
	errA := errors.New("writer A failed")
	a := &captureWriter{err: errA}
	b := &captureWriter{}
	m := &MultiWriter{Writers: []DumpWriter{a, b}}

	err := m.Write(context.Background(), DumpEntry{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errA) && err.Error() == "" {
		t.Errorf("unexpected error text: %v", err)
	}
	// b still received the entry despite a's error.
	if len(b.entries) != 1 {
		t.Errorf("b should still have received 1 entry, got %d", len(b.entries))
	}
}

func TestMultiWriterAllErrors(t *testing.T) {
	a := &captureWriter{err: errors.New("err-a")}
	b := &captureWriter{err: errors.New("err-b")}
	m := &MultiWriter{Writers: []DumpWriter{a, b}}

	err := m.Write(context.Background(), DumpEntry{})
	if err == nil {
		t.Fatal("expected error when all writers fail")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AsyncWriter
// ─────────────────────────────────────────────────────────────────────────────

func TestAsyncWriterDrainsOnClose(t *testing.T) {
	inner := &captureWriter{}
	aw := NewAsyncWriter(inner, 64)

	const n = 10
	for range n {
		_ = aw.Write(context.Background(), DumpEntry{Meta: DumpMeta{Method: "GET"}})
	}
	aw.Close() // must drain before returning

	if len(inner.entries) != n {
		t.Errorf("drained %d entries, want %d", len(inner.entries), n)
	}
}

func TestAsyncWriterDefaultQueueSize(t *testing.T) {
	// queueSize <= 0 → default 1024; just verify it doesn't panic.
	aw := NewAsyncWriter(NoopWriter{}, 0)
	_ = aw.Write(context.Background(), DumpEntry{})
	aw.Close()
}

func TestAsyncWriterDropsOnFullQueue(t *testing.T) {
	// Use a writer that blocks forever to fill the queue quickly.
	block := make(chan struct{})
	blockingWriter := DumpWriter(writerFunc(func(_ context.Context, _ DumpEntry) error {
		<-block
		return nil
	}))

	const queueSize = 4
	aw := NewAsyncWriter(blockingWriter, queueSize)

	// Enqueue more than queueSize entries; extras should be silently dropped.
	for range queueSize + 10 {
		_ = aw.Write(context.Background(), DumpEntry{})
	}

	// Unblock the writer and drain.
	close(block)
	aw.Close()
	// No assertion on count — just confirming no deadlock or panic.
}

func TestAsyncWriterContextValuesPreserved(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-abc")

	var gotCtx context.Context
	inner := DumpWriter(writerFunc(func(c context.Context, _ DumpEntry) error {
		gotCtx = c
		return nil
	}))

	aw := NewAsyncWriter(inner, 8)
	_ = aw.Write(ctx, DumpEntry{})
	aw.Close()

	if v, ok := gotCtx.Value(ctxKey{}).(string); !ok || v != "trace-abc" {
		t.Errorf("context value not preserved: %v", v)
	}
}

// writerFunc adapts a function to DumpWriter for test use.
type writerFunc func(context.Context, DumpEntry) error

func (f writerFunc) Write(ctx context.Context, e DumpEntry) error { return f(ctx, e) }
