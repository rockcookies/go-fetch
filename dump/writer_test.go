package dump

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestMultiWriter(t *testing.T) {
	tests := []struct {
		name      string
		writers   []DumpWriter
		wantErr   bool
		wantCalls int
	}{
		{
			name: "all_ok",
			writers: []DumpWriter{
				&captureWriter{},
				&captureWriter{},
			},
			wantCalls: 2,
		},
		{
			name: "one_fails",
			writers: []DumpWriter{
				&captureWriter{},
				errWriter{err: errors.New("write failed")},
			},
			wantErr:   true,
			wantCalls: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := MultiWriter{Writers: tt.writers}
			err := m.Write(context.Background(), DumpEntry{})
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type errWriter struct{ err error }

func (e errWriter) Write(context.Context, DumpEntry) error { return e.err }

func TestIOWriter(t *testing.T) {
	var buf bytes.Buffer
	w := IOWriter{W: &buf}
	entry := DumpEntry{
		Meta: DumpMeta{
			Method:  "GET",
			URL:     "http://example.com/",
			Status:  200,
			Latency: 0,
		},
		ReqHeaders: http.Header{"X-Test": {"value"}},
	}
	if err := w.Write(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "GET") || !strings.Contains(out, "X-Test") {
		t.Errorf("unexpected output: %s", out)
	}
}
