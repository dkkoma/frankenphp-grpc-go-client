package registry

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrInvalidHandle = errors.New("frankengrpc: invalid handle")
	nextHandle       atomic.Uint64
	mu               sync.RWMutex
	values           = map[uint64]any{}
)

func Put(value any) uint64 {
	handle := nextHandle.Add(1)
	mu.Lock()
	values[handle] = value
	mu.Unlock()
	return handle
}

func Get[T any](handle uint64) (T, error) {
	var zero T

	mu.RLock()
	value, ok := values[handle]
	mu.RUnlock()
	if !ok {
		return zero, ErrInvalidHandle
	}

	typed, ok := value.(T)
	if !ok {
		return zero, ErrInvalidHandle
	}

	return typed, nil
}

func Delete(handle uint64) {
	mu.Lock()
	delete(values, handle)
	mu.Unlock()
}
