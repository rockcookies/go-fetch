// Package utils provides internal utility functions.
package utils

import "context"

// ContextKey is a type-safe context key for storing and retrieving values.
type ContextKey[T any] struct {
	name string
}

// NewContextKey creates a new type-safe context key with the given name.
func NewContextKey[T any](name string) ContextKey[T] {
	return ContextKey[T]{name: name}
}

// WithValue returns a new context with the value associated with this key.
func (k *ContextKey[T]) WithValue(ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, k, value)
}

// GetValue retrieves the value associated with this key from the context.
// Returns the value and true if found, or zero value and false otherwise.
func (k *ContextKey[T]) GetValue(ctx context.Context) (res T, ok bool) {
	if ctx == nil {
		return
	}

	val := ctx.Value(k)
	if val == nil {
		return
	}
	res, ok = val.(T)
	return
}
