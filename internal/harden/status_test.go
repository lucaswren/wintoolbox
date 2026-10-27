package harden

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func mockStatusReaders(t *testing.T) {
	t.Helper()
	ps, service, registry, anonymous := runStatusPS, readWinRMService, readSMB1Registry, readAnonymousStatus
	t.Cleanup(func() {
		runStatusPS, readWinRMService, readSMB1Registry, readAnonymousStatus = ps, service, registry, anonymous
	})
	readWinRMService = func() (bool, bool, error) { return true, true, nil }
	readSMB1Registry = func() (bool, bool, string) { return true, false, "SMBv1 注册表已禁用" }
	readAnonymousStatus = func() (bool, bool, string) { return true, false, "anonymous local result" }
}

func TestStatusCancellationRetainsLocalResults(t *testing.T) {
	mockStatusReaders(t)
	var calls atomic.Int32
	runStatusPS = func(ctx context.Context, _ string) (string, error) {
		calls.Add(1)
		<-ctx.Done()
		return "", ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	status := GetStatusContext(ctx)
	if time.Since(started) > time.Second || calls.Load() != 2 {
		t.Fatalf("unbounded or repeated queries: duration=%s calls=%d", time.Since(started), calls.Load())
	}
	if !status.AnonymousOK || status.AnonymousUnknown || !status.Smb1Disabled || status.Smb1Unknown {
		t.Fatalf("local results lost on provider timeout: %+v", status)
	}
	if !status.WinRMUnknown || status.WinRMHardened || !strings.Contains(status.WinRMDetail, "deadline") {
		t.Fatalf("timeout must remain unknown, not unhardened: %+v", status)
	}
}

func TestWinRMStatusErrorsAndPartialRules(t *testing.T) {
	mockStatusReaders(t)
	for _, tc := range []struct {
		name, output string
		err          error
		hardened     bool
		unknown      bool
	}{
		{"both rules", `{"port5985":true,"port5986":true}`, nil, true, false},
		{"partial", `{"port5985":true,"port5986":false}`, nil, false, false},
		{"absent", `{"port5985":false,"port5986":false}`, nil, false, false},
		{"incomplete", `{"port5985":true}`, nil, false, true},
		{"invalid", `not JSON`, nil, false, true},
		{"access denied", "", fmt.Errorf("access denied"), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runStatusPS = func(context.Context, string) (string, error) { return tc.output, tc.err }
			hardened, unknown, detail := winrmStateContext(context.Background())
			if hardened != tc.hardened || unknown != tc.unknown {
				t.Fatalf("hardened=%v unknown=%v detail=%s", hardened, unknown, detail)
			}
		})
	}
	runStatusPS = func(context.Context, string) (string, error) { return `{"port5985":true,"port5986":true}`, nil }
	readWinRMService = func() (bool, bool, error) { return false, false, fmt.Errorf("service access denied") }
	if hardened, unknown, _ := winrmStateContext(context.Background()); hardened || !unknown {
		t.Fatal("service query failure must stay unknown")
	}
	readWinRMService = func() (bool, bool, error) { return false, true, nil }
	if hardened, unknown, _ := winrmStateContext(context.Background()); hardened || unknown {
		t.Fatal("stopped but enabled service is not fully hardened")
	}
}

func TestWinRMFirewallScript(t *testing.T) {
	dir, err := os.MkdirTemp(".", ".harden-test-")
	if err != nil {
		t.Fatal(err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	script := filepath.Join(dir, "status.ps1")
	if err := os.WriteFile(script, []byte(winRMStatusScript), 0600); err != nil {
		t.Fatal(err)
	}
	harness, err := filepath.Abs("testdata/status-tests.ps1")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", harness, "-Script", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
}
