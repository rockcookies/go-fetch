package pool

import (
	"sync"
	"testing"
)

func TestPoolGetPut(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() *int
		validate func(t *testing.T, val *int)
	}{
		{
			name: "basic int pointer",
			factory: func() *int {
				v := 42
				return &v
			},
			validate: func(t *testing.T, val *int) {
				if val == nil {
					t.Fatal("expected non-nil value")
				}
				if *val != 42 {
					t.Errorf("expected 42, got %d", *val)
				}
			},
		},
		{
			name: "zero value",
			factory: func() *int {
				v := 0
				return &v
			},
			validate: func(t *testing.T, val *int) {
				if val == nil {
					t.Fatal("expected non-nil value")
				}
				if *val != 0 {
					t.Errorf("expected 0, got %d", *val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.factory)

			// Get from pool (should call factory first time)
			val := p.Get()
			tt.validate(t, val)

			// Put back to pool
			p.Put(val)

			// Get again (should reuse from pool)
			val2 := p.Get()
			tt.validate(t, val2)

			// Verify it's the same object
			if val != val2 {
				t.Error("expected to reuse same object from pool")
			}
		})
	}
}

func TestPoolConcurrency(t *testing.T) {
	p := New(func() *int {
		v := 0
		return &v
	})

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				val := p.Get()
				if val == nil {
					t.Error("got nil from pool")
					return
				}
				*val = j
				p.Put(val)
			}
		}()
	}

	wg.Wait()
}

func TestPoolMultipleGets(t *testing.T) {
	callCount := 0
	p := New(func() *int {
		callCount++
		v := callCount
		return &v
	})

	// Get multiple items without putting back
	val1 := p.Get()
	val2 := p.Get()
	val3 := p.Get()

	// Each should be different since pool is empty
	if val1 == val2 || val2 == val3 || val1 == val3 {
		t.Error("expected different objects when pool is empty")
	}

	if *val1 != 1 || *val2 != 2 || *val3 != 3 {
		t.Errorf("unexpected values: %d, %d, %d", *val1, *val2, *val3)
	}

	// Put them back
	p.Put(val1)
	p.Put(val2)
	p.Put(val3)

	// Get again - should reuse
	val4 := p.Get()
	val5 := p.Get()
	val6 := p.Get()

	// Should be one of the original values
	found := false
	for _, v := range []*int{val1, val2, val3} {
		if v == val4 || v == val5 || v == val6 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to reuse objects from pool")
	}
}

func TestPoolWithStruct(t *testing.T) {
	type testStruct struct {
		Value  int
		String string
	}

	p := New(func() *testStruct {
		return &testStruct{
			Value:  42,
			String: "test",
		}
	})

	val := p.Get()
	if val.Value != 42 || val.String != "test" {
		t.Errorf("unexpected struct values: %+v", val)
	}

	// Modify and put back
	val.Value = 100
	val.String = "modified"
	p.Put(val)

	// Get again - should have modified values (pool doesn't reset)
	val2 := p.Get()
	if val2.Value != 100 || val2.String != "modified" {
		t.Errorf("expected modified values: %+v", val2)
	}
}

func BenchmarkPool(b *testing.B) {
	p := New(func() *int {
		v := 0
		return &v
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val := p.Get()
			*val++
			p.Put(val)
		}
	})
}

func BenchmarkPoolWithoutReuse(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v := 0
			val := &v
			*val++
		}
	})
}
