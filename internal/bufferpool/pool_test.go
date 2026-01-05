package bufferpool

import (
	"bytes"
	"sync"
	"testing"
)

func TestGetPut(t *testing.T) {
	tests := []struct {
		name        string
		initialData string
		writeData   string
		wantReset   bool
	}{
		{
			name:        "get returns reset buffer",
			initialData: "",
			writeData:   "test",
			wantReset:   true,
		},
		{
			name:        "buffer can be written to",
			initialData: "",
			writeData:   "hello world",
			wantReset:   true,
		},
		{
			name:        "empty string write",
			initialData: "",
			writeData:   "",
			wantReset:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := Get()
			if buf == nil {
				t.Fatal("Get() returned nil")
			}

			// Buffer should be empty after Get()
			if buf.Len() != 0 {
				t.Errorf("Get() returned non-empty buffer, len=%d", buf.Len())
			}

			// Write data
			buf.WriteString(tt.writeData)
			if buf.String() != tt.writeData {
				t.Errorf("buffer content = %q, want %q", buf.String(), tt.writeData)
			}

			// Put back
			Put(buf)

			// Get again - should be reset
			buf2 := Get()
			if tt.wantReset && buf2.Len() != 0 {
				t.Errorf("buffer not reset after Put/Get, len=%d", buf2.Len())
			}

			Put(buf2)
		})
	}
}

func TestGetReturnsResetBuffer(t *testing.T) {
	// Get a buffer and write to it
	buf := Get()
	buf.WriteString("some data that should be cleared")

	// Put it back
	Put(buf)

	// Get should return a reset buffer
	buf2 := Get()
	if buf2.Len() != 0 {
		t.Errorf("expected empty buffer, got len=%d, content=%q", buf2.Len(), buf2.String())
	}

	Put(buf2)
}

func TestBufferReuse(t *testing.T) {
	buf1 := Get()
	buf1.WriteString("test")
	Put(buf1)

	buf2 := Get()

	// buf2 should be the same underlying buffer (reused)
	// This is implementation detail but good to verify
	if buf1 != buf2 {
		t.Log("warning: buffer not reused (this is ok, just less efficient)")
	}

	// But it should be reset
	if buf2.Len() != 0 {
		t.Errorf("reused buffer not reset, len=%d", buf2.Len())
	}

	Put(buf2)
}

func TestConcurrency(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := Get()
				if buf == nil {
					t.Error("Get() returned nil")
					return
				}

				// Verify buffer is empty
				if buf.Len() != 0 {
					t.Errorf("goroutine %d: got non-empty buffer, len=%d", id, buf.Len())
					return
				}

				// Use the buffer
				buf.WriteString("test")

				// Return to pool
				Put(buf)
			}
		}(i)
	}

	wg.Wait()
}

func TestBufferCapacityGrowth(t *testing.T) {
	buf := Get()

	// Write a large amount of data
	largeData := string(make([]byte, 10000))
	buf.WriteString(largeData)

	initialCap := buf.Cap()
	if initialCap < 10000 {
		t.Errorf("buffer capacity %d is less than written data size 10000", initialCap)
	}

	Put(buf)

	// Get again - should maintain capacity
	buf2 := Get()
	if buf2.Cap() < initialCap {
		t.Errorf("buffer capacity decreased: %d -> %d", initialCap, buf2.Cap())
	}

	Put(buf2)
}

func TestMultipleGetsPuts(t *testing.T) {
	buffers := make([]*bytes.Buffer, 10)

	// Get multiple buffers
	for i := range buffers {
		buffers[i] = Get()
		if buffers[i] == nil {
			t.Fatalf("Get() returned nil at index %d", i)
		}
	}

	// Use them
	for i, buf := range buffers {
		buf.WriteString("buffer")
		buf.WriteString(string(rune('0' + i)))
	}

	// Put them back
	for _, buf := range buffers {
		Put(buf)
	}

	// Get them again - should be reset
	for i := 0; i < 10; i++ {
		buf := Get()
		if buf.Len() != 0 {
			t.Errorf("buffer %d not reset, len=%d", i, buf.Len())
		}
		Put(buf)
	}
}

func BenchmarkGetPut(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := Get()
			buf.WriteString("test data")
			Put(buf)
		}
	})
}

func BenchmarkGetPutLarge(b *testing.B) {
	data := string(make([]byte, 1024))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := Get()
			buf.WriteString(data)
			Put(buf)
		}
	})
}

func BenchmarkWithoutPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := &bytes.Buffer{}
			buf.WriteString("test data")
		}
	})
}
