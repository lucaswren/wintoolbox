package selfupdate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadStateResponsiveAndCancellationCleansPartial(t *testing.T) {
	dir := t.TempDir()
	previousDirectory, previousInfo := cacheDirectory, Cached()
	cacheDirectory = func() (string, error) { return dir, nil }
	t.Cleanup(func() { cacheDirectory = previousDirectory; setCached(previousInfo) })
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(entered)
		<-r.Context().Done()
	}))
	defer server.Close()
	setCached(Info{CurrentVersion: "1.0.0", LatestVersion: "2.0.0", HasUpdate: true, AssetName: "test.exe", AssetURL: server.URL, AssetSHA256: strings.Repeat("a", 64), AssetSize: 1024})
	done := make(chan error, 1)
	go func() { _, err := Download("1.0.0"); done <- err }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("download did not start")
	}
	state := make(chan Info, 1)
	go func() { state <- Cached() }()
	select {
	case info := <-state:
		if !info.Downloading {
			t.Fatal("missing live download state")
		}
	case <-time.After(time.Second):
		t.Fatal("state read blocked by transfer")
	}
	CancelDownload()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancellation ignored")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("download did not cancel")
	}
	if info := Cached(); info.Downloading || info.Downloaded || info.Error == "" {
		t.Fatalf("incorrect final state: %+v", info)
	}
	partial := downloadPath("2.0.0", "test.exe") + ".partial"
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("partial download retained")
	}
}

// Exercise the actual handoff script using temporary files and a fake launcher.
// No installed executable or Windows configuration is touched.
func TestUpdaterRollsBackFailedLaunch(t *testing.T) {
	dir := t.TempDir()
	target, source := filepath.Join(dir, "app.exe"), filepath.Join(dir, "package.exe")
	if err := os.WriteFile(target, []byte("old executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("new executable"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := fileSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	script := "function Start-Process { throw 'injected launcher failure' }\n" + updateScript(target, source, hash, filepath.Join(dir, "install.log"), 2147483647)
	path := filepath.Join(dir, "apply.ps1")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput(); err == nil {
		t.Fatalf("expected failure: %s", out)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old executable" {
		t.Fatalf("rollback failed: %q", data)
	}
	log, err := os.ReadFile(filepath.Join(dir, "install.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "Restored previous executable") {
		t.Fatal(string(log))
	}
}

func TestManifestMatchesSelectedAsset(t *testing.T) {
	first, second := strings.Repeat("a", 64), strings.Repeat("b", 64)
	manifest := first + " *other-arm64.exe\n" + second + " *WinToolbox.exe"
	if got := parseSHA256ForAsset(manifest, "WinToolbox.exe"); got != second {
		t.Fatalf("wrong asset: %s", got)
	}
	if got := parseSHA256ForAsset(manifest, "missing.exe"); got != "" {
		t.Fatalf("unmatched asset accepted: %s", got)
	}
	if got := parseSHA256ForAsset(first, "WinToolbox.exe"); got != first {
		t.Fatal("hash-only sidecar rejected")
	}
	if digestSHA256("sha256:bad") != "" {
		t.Fatal("malformed digest accepted")
	}
}
func TestPrereleaseOrdering(t *testing.T) {
	for _, pair := range [][2]string{{"1.2.0", "1.2.0-rc.1"}, {"1.2.0-rc.10", "1.2.0-rc.2"}, {"1.3.0-beta", "1.2.9"}} {
		if compareVersions(pair[0], pair[1]) <= 0 {
			t.Fatal(pair)
		}
	}
	if compareVersions("1.2.0+build", "v1.2.0") != 0 {
		t.Fatal("metadata affects ordering")
	}
}
