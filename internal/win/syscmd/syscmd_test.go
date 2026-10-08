package syscmd

import (
	"context"
	"errors"
	"testing"
)

func TestCancelledReadDoesNotLaunchCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RunQuickContext(ctx, "does-not-exist.exe")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
