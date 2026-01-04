package utils

import "context"

type ContextKey[T any] struct {
	name string
}

func NewContextKey[T any](name string) ContextKey[T] {
	return ContextKey[T]{name: name}
}

func (k *ContextKey[T]) WithValue(ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, k, value)
}

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
