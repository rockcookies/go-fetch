// Package dump provides a production-grade HTTP dump transport.
//
// # Architecture
//
// DumpTransport wraps any http.RoundTripper and intercepts every HTTP exchange.
// It is safe for concurrent use.
//
// # Filtering
//
// Two filter hooks control whether an exchange is dumped:
//   - PreFilter  – evaluated before the request is sent (no response available).
//     Use it to skip well-known noisy endpoints early and avoid reading the
//     request body unnecessarily.
//   - PostFilter – evaluated after the response headers arrive (resp is non-nil
//     on success, nil on transport error).  Use it to match by status code.
//
// Both filters receive (req, resp, err).  In PreFilter, resp and err are always
// nil.  In PostFilter, resp is nil when a transport error occurred.
//
// # Panic handling
//
// If the inner RoundTripper panics, DumpTransport will:
//  1. recover() inside an isolated closure (keeping the outer stack clean),
//  2. flush a dump entry with PanicInfo so the event is observable,
//  3. re-panic with the original value, honouring the RoundTripper contract.
//
// # Streaming response capture
//
// When DumpResponseBody is set, the response body is wrapped with a tee reader
// so callers consume bytes normally; the dump is written only when the body is
// fully closed.  This avoids buffering the entire response in memory.
//
// # Async writing
//
// Wrap any DumpWriter with NewAsyncWriter to offload I/O from the hot path.
// Entries dropped when the queue is full are silently discarded (back-pressure
// avoidance).  Call Close() during shutdown to drain the queue.
package dump
