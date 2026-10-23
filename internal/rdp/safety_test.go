package rdp

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeSources(t *testing.T) {
	got, err := NormalizeSources("192.0.2.8;192.0.2.8/32\n2001:db8::123/64 198.51.100.9/24")
	want := []string{"192.0.2.8/32", "2001:db8::/64", "198.51.100.0/24"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%v, %v", got, err)
	}
	for _, value := range []string{"", "Any", "*", "example.com", "0.0.0.0/0", "::/0", "0.0.0.0", "::", "224.0.0.1", "::ffff:192.0.2.1", "fe80::1%3", "192.0.2.8';whoami"} {
		if _, err := NormalizeSources(value); err == nil {
			t.Errorf("accepted unsafe input %q", value)
		}
	}
}

// Exercise the real PowerShell transaction flow with isolated filesystem state
// and mocked Windows providers. No host firewall, registry or tasks are changed.
func TestSafetyTransactions(t *testing.T) {
	// Keep subprocess fixtures in the workspace: some restricted Windows hosts
	// deny child PowerShell atomic renames in Go's private system TEMP folders.
	dir, err := os.MkdirTemp(".", ".rdp-safety-test-")
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
	script := filepath.Join(dir, "safety.ps1")
	if err := os.WriteFile(script, []byte(safetyScript), 0600); err != nil {
		t.Fatal(err)
	}
	harness, err := filepath.Abs("testdata/safety-tests.ps1")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", harness, "-Script", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
}
