package rdp

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNativeNetworkMutexSerializesThreads(t *testing.T) {
	name := fmt.Sprintf(`Local\WinToolBox-Test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	unlock, err := lockNetworkMutationNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			unlock()
		}
	})
	done := make(chan error, 1)
	go func() {
		release, err := lockNetworkMutationNamed(name)
		if err == nil {
			release()
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("second writer did not wait: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	unlock()
	released = true
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not acquire after release")
	}
}
