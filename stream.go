package fetch

import (
	"io"
	"time"
)

// callbackWriter wraps an io.Writer to invoke a callback periodically during writes.
// Used internally for progress tracking during multipart uploads.
//
// The callback is invoked either when the total size is reached or at regular intervals
// specified by interval. This prevents excessive callback invocations for large uploads
// while still providing responsive progress updates.
type callbackWriter struct {
	io.Writer
	written   int64
	totalSize int64
	lastTime  time.Time
	interval  time.Duration
	callback  func(written int64)
}

func (w *callbackWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	if n <= 0 {
		return
	}
	w.written += int64(n)
	if w.written == w.totalSize {
		w.callback(w.written)
	} else if now := time.Now(); now.Sub(w.lastTime) >= w.interval {
		w.lastTime = now
		w.callback(w.written)
	}
	return
}
