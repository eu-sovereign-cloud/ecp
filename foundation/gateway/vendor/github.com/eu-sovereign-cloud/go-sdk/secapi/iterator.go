package secapi

import (
	"context"
	"errors"
	"io"
)

// IteratorFunc is a function that can be used to fetch a collection of items.
type IteratorFunc[T any] func(ctx context.Context, skipToken *string) ([]T, *string, error)

// Iterator is a generic iterator that can be used to iterate over a collection of items.
type Iterator[T any] struct {
	fn        IteratorFunc[T]
	ptr       int
	buf       []T
	skipToken *string
}

// NewIterator creates a new iterator.
func NewIterator[T any](fn IteratorFunc[T]) *Iterator[T] {
	return &Iterator[T]{
		fn: fn,
	}
}

// Next returns the next item in the iterator. If there are no more items, it returns io.EOF.
func (i *Iterator[T]) Next(ctx context.Context) (*T, error) {
	// If we have items in the buffer, return the next one
	if i.ptr < len(i.buf) {
		result := &i.buf[i.ptr]
		i.ptr++
		return result, nil
	}

	// If we don't have a skip token and the buffer is empty, we're done
	if i.skipToken == nil && len(i.buf) > 0 {
		return nil, io.EOF
	}

	// Fetch more items
	newData, newSkipToken, err := i.fn(ctx, i.skipToken)
	if err != nil {
		return nil, err
	}

	// Update the buffer and skip token
	i.buf = newData
	i.skipToken = newSkipToken

	// If we have items in the buffer, return the first one
	if len(i.buf) > 0 {
		result := &i.buf[0]
		i.ptr = 1
		return result, nil
	}

	return nil, io.EOF
}

// All returns all items in the iterator.
func (i *Iterator[T]) All(ctx context.Context) ([]*T, error) {
	var items []*T
	for {
		item, err := i.Next(context.Background())
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
