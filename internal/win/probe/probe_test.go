package probe

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimedOutProbeIsReused(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var calls atomic.Int32
	fn := func() int { calls.Add(1); close(started); <-release; return 42 }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := Do(ctx, "test-coalescing", fn); err == nil {
		t.Fatal("expected timeout")
	}
	<-started
	second, cancelSecond := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelSecond()
	if _, err := Do(second, "test-coalescing", fn); err == nil {
		t.Fatal("expected second caller timeout")
	}
	mu.Lock()
	done := jobs["test-coalescing"].done
	mu.Unlock()
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("caller did not finish")
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate slow native probe")
	}
}

func TestSharedContextCancelledWhenCallerLeaves(t *testing.T) {
	completed := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := DoContext(ctx, "test-cancel", func(shared context.Context) int { <-shared.Done(); close(completed); return 0 })
	if err == nil {
		t.Fatal("expected caller timeout")
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("command context was not cancelled")
	}
}
