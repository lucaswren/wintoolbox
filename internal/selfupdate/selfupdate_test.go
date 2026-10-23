package selfupdate

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type cancelBody struct {
	ctx     context.Context
	entered chan struct{}
	started bool
}

func (b *cancelBody) Read(p []byte) (int, error) {
	if !b.started {
		b.started = true
		close(b.entered)
		return copy(p, "partial"), nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}
func (b *cancelBody) Close() error { return nil }
func workspaceTemp(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".selfupdate-test-")
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
	return dir
}

func TestDownloadStateResponsiveAndCancellationCleansPartial(t *testing.T) {
	dir := t.TempDir()
	previousDirectory, previousInfo := cacheDirectory, Cached()
	cacheDirectory = func() (string, error) { return dir, nil }
	t.Cleanup(func() { cacheDirectory = previousDirectory; setCached(previousInfo) })
	entered := make(chan struct{})
	previousClient := downloadHTTPClient
	downloadHTTPClient = func() *http.Client {
		return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &cancelBody{ctx: r.Context(), entered: entered}}, nil
		})}
	}
	t.Cleanup(func() { downloadHTTPClient = previousClient })
	setCached(Info{CurrentVersion: "1.0.0", LatestVersion: "2.0.0", HasUpdate: true, AssetName: "test.exe", AssetURL: "https://update.test/package.exe", AssetSHA256: strings.Repeat("a", 64), AssetSize: 1024})
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() { defer close(finished); _, err := Download("1.0.0"); done <- err }()
	t.Cleanup(func() {
		CancelDownload()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("download worker did not stop")
		}
	})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("download did not start")
	case err := <-done:
		t.Fatalf("download exited before body read: %v", err)
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
	dir := workspaceTemp(t)
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
	if !strings.Contains(string(log), "injected launcher failure") {
		t.Fatal("replacement did not reach launch failure", string(log))
	}
}

func TestDownloadTransportFailureClearsState(t *testing.T) {
	oldClient, oldDirectory, oldInfo := downloadHTTPClient, cacheDirectory, Cached()
	t.Cleanup(func() { downloadHTTPClient = oldClient; cacheDirectory = oldDirectory; setCached(oldInfo) })
	dir := t.TempDir()
	cacheDirectory = func() (string, error) { return dir, nil }
	downloadHTTPClient = func() *http.Client {
		return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}
	}
	setCached(Info{CurrentVersion: "1.0.0", LatestVersion: "2.0.0", HasUpdate: true, AssetName: "test.exe", AssetURL: "https://update.test/fail", AssetSHA256: strings.Repeat("a", 64)})
	if _, err := Download("1.0.0"); err == nil {
		t.Fatal("transport failure ignored")
	}
	if state := Cached(); state.Downloading || state.Downloaded || state.Error == "" {
		t.Fatalf("stale download state: %+v", state)
	}
}

func TestDownloadRejectsCorruptPackageAndRemovesPartial(t *testing.T) {
	oldClient, oldDirectory, oldInfo := downloadHTTPClient, cacheDirectory, Cached()
	t.Cleanup(func() { downloadHTTPClient = oldClient; cacheDirectory = oldDirectory; setCached(oldInfo) })
	dir := t.TempDir()
	cacheDirectory = func() (string, error) { return dir, nil }
	downloadHTTPClient = func() *http.Client {
		return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("corrupt"))}, nil
		})}
	}
	setCached(Info{CurrentVersion: "1.0.0", LatestVersion: "2.0.0", HasUpdate: true, AssetName: "test.exe", AssetURL: "https://update.test/corrupt", AssetSHA256: strings.Repeat("a", 64), AssetSize: 7})
	if _, err := Download("1.0.0"); err == nil {
		t.Fatal("corrupt package accepted")
	}
	for _, p := range []string{downloadPath("2.0.0", "test.exe"), downloadPath("2.0.0", "test.exe") + ".partial"} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("invalid package retained: %s, %v", p, err)
		}
	}
	if s := Cached(); s.Downloading || s.Verified || s.Downloaded || s.Error == "" {
		t.Fatalf("incorrect failure state: %+v", s)
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

func TestChecksumsDoNotMatchOtherExecutables(t *testing.T) {
	hash := strings.Repeat("a", 64)
	for _, text := range []string{
		"OtherWinToolbox.exe SHA256: " + hash,
		"WinToolbox.exe.sha256 SHA256: " + hash,
		"arm64.exe SHA256: " + hash,
		"SHA256: " + hash + "\nSHA256: " + strings.Repeat("b", 64),
	} {
		if got := parseSHA256FromNotes(text, "WinToolbox.exe"); got != "" {
			t.Fatalf("unmatched checksum accepted for %q: %s", text, got)
		}
		if got := parseSHA256ForAsset(text, "WinToolbox.exe"); got != "" {
			t.Fatalf("unmatched manifest entry accepted for %q: %s", text, got)
		}
	}
	for _, text := range []string{"`WinToolbox.exe` SHA256: " + hash, hash + " *WinToolbox.exe", "SHA256: " + hash} {
		if got := parseSHA256FromNotes(text, "WinToolbox.exe"); got != hash {
			t.Fatalf("valid checksum rejected for %q: %s", text, got)
		}
	}
}
