package bufferpool

import (
	"bytes"

	"github.com/rockcookies/go-fetch/internal/pool"
)

type Pool struct {
	p *pool.Pool[*bytes.Buffer]
}

var _pool = &Pool{
	p: pool.New(func() *bytes.Buffer {
		return &bytes.Buffer{}
	}),
}

func Get() *bytes.Buffer {
	buf := _pool.p.Get()
	buf.Reset()
	return buf
}

func Put(buf *bytes.Buffer) {
	_pool.p.Put(buf)
}
