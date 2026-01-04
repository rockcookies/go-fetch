// Package bufferpool provides a buffer pool for bytes.Buffer instances.
package bufferpool

import (
	"bytes"

	"github.com/rockcookies/go-fetch/internal/pool"
)

// Pool wraps a generic pool for bytes.Buffer.
type Pool struct {
	p *pool.Pool[*bytes.Buffer]
}

var _pool = &Pool{
	p: pool.New(func() *bytes.Buffer {
		return &bytes.Buffer{}
	}),
}

// Get retrieves a buffer from the pool and resets it.
func Get() *bytes.Buffer {
	buf := _pool.p.Get()
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool for reuse.
func Put(buf *bytes.Buffer) {
	_pool.p.Put(buf)
}
