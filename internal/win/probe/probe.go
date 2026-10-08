// Package probe coalesces overlapping reads without allowing timed-out callers
// to accumulate new copies of a slow native operation.
package probe

import (
	"context"
	"fmt"
	"sync"
)

type job struct {
	done    chan struct{}
	value   any
	err     error
	waiters int
	cancel  context.CancelFunc
}

var mu sync.Mutex
var jobs = map[string]*job{}

func Do[T any](ctx context.Context, key string, fn func() T) (T, error) {
	return DoContext(ctx, key, func(context.Context) T { return fn() })
}

// DoContext cancels the shared command only when every waiting caller leaves.
func DoContext[T any](ctx context.Context, key string, fn func(context.Context) T) (T, error) {
	mu.Lock()
	j := jobs[key]
	if j == nil {
		shared, cancel := context.WithCancel(context.Background())
		j = &job{done: make(chan struct{}), cancel: cancel}
		jobs[key] = j
		go func() {
			defer func() {
				cancel()
				if r := recover(); r != nil {
					j.err = fmt.Errorf("状态探测异常: %v", r)
				}
				mu.Lock()
				delete(jobs, key)
				close(j.done)
				mu.Unlock()
			}()
			j.value = fn(shared)
		}()
	}
	j.waiters++
	mu.Unlock()
	defer func() {
		mu.Lock()
		j.waiters--
		if j.waiters == 0 {
			j.cancel()
		}
		mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-j.done:
		if j.err != nil {
			var zero T
			return zero, j.err
		}
		return j.value.(T), nil
	}
}
