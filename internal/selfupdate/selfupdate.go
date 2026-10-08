package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"wintoolbox/internal/win/syscmd"
)

const (
	repoOwner = "lucaswren"
	repoName  = "wintoolbox"
	userAgent = "WinToolbox-SelfUpdate"
)

var cacheDirectory = os.UserCacheDir

// Info describes the latest release relative to the running app.
type Info struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	HasUpdate       bool   `json:"hasUpdate"`
	AssetName       string `json:"assetName"`
	AssetURL        string `json:"assetURL"`
	AssetSize       int64  `json:"assetSize"`
	AssetSHA256     string `json:"assetSHA256"`
	Notes           string `json:"notes"`
	Downloaded      bool   `json:"downloaded"`
	DownloadPath    string `json:"downloadPath"`
	Verified        bool   `json:"verified"`
	Downloading     bool   `json:"downloading"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	BytesPerSecond  int64  `json:"bytesPerSecond"`
	Error           string `json:"error,omitempty"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Body    string    `json:"body"`
	Assets  []ghAsset `json:"assets"`
}

var (
	operationMu    sync.Mutex
	cancelDownload context.CancelFunc
	mu             sync.Mutex
	cached         Info

	sha256BodyRe = regexp.MustCompile(`(?i)SHA256\s*[:：]?\s*` + "`?" + `([A-Fa-f0-9]{64})` + "`?")
)

// Check queries GitHub Releases for a newer version than current.
func Check(current string) (Info, error) {
	operationMu.Lock()
	defer operationMu.Unlock()
	return checkLocked(current)
}

func checkLocked(current string) (Info, error) {
	info := Info{CurrentVersion: normalizeVersion(current)}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		info.Error = err.Error()
		setCached(info)
		return info, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		info.Error = "无法连接更新服务器: " + err.Error()
		setCached(info)
		return info, fmt.Errorf("%s", info.Error)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		info.Error = "读取更新信息失败: " + err.Error()
		setCached(info)
		return info, fmt.Errorf("%s", info.Error)
	}
	if resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("更新接口返回 %d", resp.StatusCode)
		setCached(info)
		return info, fmt.Errorf("%s", info.Error)
	}

	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		info.Error = "解析更新信息失败"
		setCached(info)
		return info, fmt.Errorf("%s: %w", info.Error, err)
	}

	info.LatestVersion = normalizeVersion(rel.TagName)
	info.Notes = strings.TrimSpace(rel.Body)
	info.HasUpdate = compareVersions(info.LatestVersion, info.CurrentVersion) > 0

	asset := pickAsset(rel.Assets)
	if asset != nil {
		info.AssetName = asset.Name
		info.AssetURL = asset.BrowserDownloadURL
		info.AssetSize = asset.Size
		info.AssetSHA256 = resolveAssetSHA256(rel.Assets, asset, rel.Body)
	} else if info.HasUpdate {
		info.Error = "最新版本未找到 Windows amd64 安装包"
		setCached(info)
		return info, fmt.Errorf("%s", info.Error)
	}

	if info.HasUpdate && info.AssetName != "" {
		dest := downloadPath(info.LatestVersion, info.AssetName)
		if ok, verified := fileReadyVerified(dest, info.AssetSize, info.AssetSHA256); ok {
			info.Downloaded = true
			info.DownloadPath = dest
			info.Verified = verified
			_ = syscmd.UnblockFile(dest)
		}
	}

	setCached(info)
	return info, nil
}

// Download downloads the latest release asset when an update is available.
func Download(current string) (result Info, downloadErr error) {
	operationMu.Lock()
	defer operationMu.Unlock()
	info := Cached()
	if info.LatestVersion == "" || info.CurrentVersion == "" {
		var err error
		info, err = checkLocked(current)
		if err != nil {
			return info, err
		}
	}
	if !info.HasUpdate {
		return info, nil
	}
	if info.AssetURL == "" {
		return info, fmt.Errorf("没有可下载的更新包")
	}
	if len(info.AssetSHA256) != 64 || !isHex(info.AssetSHA256) {
		return info, fmt.Errorf("缺少有效 SHA256，已拒绝下载未校验的更新包")
	}

	dest := downloadPath(info.LatestVersion, info.AssetName)
	if ok, verified := fileReadyVerified(dest, info.AssetSize, info.AssetSHA256); ok {
		info.Downloaded = true
		info.DownloadPath = dest
		info.Verified = verified
		_ = syscmd.UnblockFile(dest)
		setCached(info)
		return info, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	mu.Lock()
	cancelDownload = cancel
	mu.Unlock()
	info.Downloading = true
	info.DownloadedBytes = 0
	info.Error = ""
	setCached(info)
	defer func() {
		cancel()
		mu.Lock()
		cancelDownload = nil
		mu.Unlock()
		info.Downloading = false
		if downloadErr != nil {
			info.Error = downloadErr.Error()
		}
		setCached(info)
		result = info
	}()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return info, fmt.Errorf("创建下载目录失败: %w", err)
	}

	tmp := dest + ".partial"
	_ = os.Remove(tmp)

	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return info, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(tmp)
	if err != nil {
		return info, fmt.Errorf("创建临时文件失败: %w", err)
	}
	h := sha256.New()
	progress := &downloadProgress{info: &info, started: time.Now()}
	written, copyErr := io.Copy(io.MultiWriter(f, h, progress), io.LimitReader(resp.Body, maxAssetSize(info.AssetSize)+1))
	if written > maxAssetSize(info.AssetSize) {
		copyErr = fmt.Errorf("更新包超过大小限制")
	}
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return info, fmt.Errorf("写入更新包失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return info, closeErr
	}
	if info.AssetSize > 0 && written != info.AssetSize {
		_ = os.Remove(tmp)
		return info, fmt.Errorf("下载不完整: 期望 %d 字节，实际 %d", info.AssetSize, written)
	}

	got := strings.ToLower(hex.EncodeToString(h.Sum(nil)))
	if info.AssetSHA256 != "" {
		want := strings.ToLower(info.AssetSHA256)
		if got != want {
			_ = os.Remove(tmp)
			return info, fmt.Errorf("更新包校验失败: SHA256 不匹配（期望 %s，实际 %s）", want, got)
		}
		info.Verified = true
	} else {
		_ = os.Remove(tmp)
		return info, fmt.Errorf("发布页未提供 SHA256，已拒绝安装未校验的更新包")
	}

	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return info, fmt.Errorf("保存更新包失败: %w", err)
	}
	_ = syscmd.UnblockFile(dest)

	info.Downloaded = true
	info.DownloadPath = dest
	setCached(info)
	return info, nil
}

// Cached returns the last Check/Download result without network I/O.
func Cached() Info {
	mu.Lock()
	defer mu.Unlock()
	return cached
}

func pickAsset(assets []ghAsset) *ghAsset {
	var named *ghAsset
	var archLoose *ghAsset
	for i := range assets {
		a := &assets[i]
		lower := strings.ToLower(a.Name)
		if !strings.HasSuffix(lower, ".exe") {
			continue
		}
		if strings.Contains(lower, "arm64") || strings.Contains(lower, "aarch64") {
			continue
		}
		if strings.Contains(lower, "windows") && (strings.Contains(lower, "amd64") || strings.Contains(lower, "x64")) {
			return a
		}
		if strings.Contains(lower, "amd64") || strings.Contains(lower, "x64") || strings.Contains(lower, "x86_64") {
			if archLoose == nil {
				archLoose = a
			}
			continue
		}
		// Only accept arch-less names that clearly belong to this product.
		if strings.HasPrefix(lower, "wintoolbox") &&
			!strings.Contains(lower, "386") &&
			!strings.Contains(lower, "i386") &&
			!strings.Contains(lower, "arm") {
			if named == nil {
				named = a
			}
		}
	}
	if archLoose != nil {
		return archLoose
	}
	return named
}

// resolveAssetSHA256 prefers GitHub digest, then release notes, then a sidecar .sha256 / SHA256.txt asset.
func resolveAssetSHA256(assets []ghAsset, exe *ghAsset, notes string) string {
	if exe == nil {
		return ""
	}
	if sum := digestSHA256(exe.Digest); sum != "" {
		return sum
	}
	if sum := parseSHA256FromNotes(notes, exe.Name); sum != "" {
		return sum
	}
	wantSidecar := strings.ToLower(exe.Name) + ".sha256"
	for i := range assets {
		a := &assets[i]
		lower := strings.ToLower(a.Name)
		if lower == wantSidecar || lower == "sha256.txt" {
			if sum, err := fetchTextSHA256(a.BrowserDownloadURL, exe.Name); err == nil && sum != "" {
				return sum
			}
		}
	}
	return ""
}

func fetchTextSHA256(url, assetName string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return "", err
	}
	return parseSHA256ForAsset(string(body), assetName), nil
}

func parseSHA256Text(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Formats: "<64hex>", "<64hex> *file", "SHA256: <64hex>"
	if sum := parseSHA256FromNotes(s, ""); sum != "" {
		return sum
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	cand := strings.TrimPrefix(strings.ToLower(fields[0]), "sha256:")
	if len(cand) == 64 && isHex(cand) {
		return cand
	}
	return ""
}

func digestSHA256(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	lower := strings.ToLower(digest)
	if strings.HasPrefix(lower, "sha256:") {
		candidate := strings.TrimPrefix(lower, "sha256:")
		if len(candidate) == 64 && isHex(candidate) {
			return candidate
		}
		return ""
	}
	if len(digest) == 64 && isHex(digest) {
		return lower
	}
	return ""
}

func parseSHA256FromNotes(notes, assetName string) string {
	if assetName != "" {
		want := strings.ToLower(filepath.Base(assetName))
		for _, line := range strings.Split(notes, "\n") {
			if !strings.Contains(strings.ToLower(line), want) {
				continue
			}
			if m := sha256BodyRe.FindStringSubmatch(line); len(m) >= 2 {
				return strings.ToLower(m[1])
			}
			for _, f := range strings.Fields(line) {
				cand := strings.TrimPrefix(strings.ToLower(f), "sha256:")
				cand = strings.Trim(cand, "`\"'")
				if len(cand) == 64 && isHex(cand) {
					return cand
				}
			}
		}
	}
	m := sha256BodyRe.FindStringSubmatch(notes)
	if len(m) >= 2 {
		return strings.ToLower(m[1])
	}
	return ""
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func downloadDir() string {
	base, err := cacheDirectory()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "WinToolbox", "updates")
}

func downloadPath(version, assetName string) string {
	safe := strings.ReplaceAll(normalizeVersion(version), "/", "_")
	return filepath.Join(downloadDir(), safe+"-"+filepath.Base(assetName))
}

func fileReadyVerified(path string, expectSize int64, expectSHA256 string) (ok bool, verified bool) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return false, false
	}
	if expectSize > 0 && st.Size() != expectSize {
		return false, false
	}
	if expectSHA256 == "" {
		return true, false
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return false, false
	}
	if !strings.EqualFold(sum, expectSHA256) {
		_ = os.Remove(path)
		return false, false
	}
	return true, true
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func compareVersions(a, b string) int {
	originalA, originalB := normalizeVersion(a), normalizeVersion(b)
	as := versionParts(a)
	bs := versionParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	// Stable releases sort after prereleases of the same core version.
	pre := func(v string) string {
		v = strings.SplitN(v, "+", 2)[0]
		parts := strings.SplitN(v, "-", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	ap, bp := pre(originalA), pre(originalB)
	if ap == bp {
		return 0
	}
	if ap == "" {
		return 1
	}
	if bp == "" {
		return -1
	}
	aa, bb := strings.Split(ap, "."), strings.Split(bp, ".")
	for i := 0; i < len(aa) && i < len(bb); i++ {
		if aa[i] == bb[i] {
			continue
		}
		an, ae := strconv.Atoi(aa[i])
		bn, be := strconv.Atoi(bb[i])
		if ae == nil && be == nil {
			if an > bn {
				return 1
			}
			return -1
		}
		if ae == nil {
			return -1
		}
		if be == nil {
			return 1
		}
		if aa[i] > bb[i] {
			return 1
		}
		return -1
	}
	if len(aa) > len(bb) {
		return 1
	}
	return -1
}

func versionParts(v string) []int {
	v = normalizeVersion(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
			for _, ch := range p {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
		}
		out = append(out, n)
	}
	return out
}

func setCached(info Info) { mu.Lock(); cached = info; mu.Unlock() }

func CancelDownload() {
	mu.Lock()
	defer mu.Unlock()
	if cancelDownload != nil {
		cancelDownload()
	}
}

type downloadProgress struct {
	info    *Info
	started time.Time
	last    time.Time
}

func (p *downloadProgress) Write(data []byte) (int, error) {
	p.info.DownloadedBytes += int64(len(data))
	seconds := time.Since(p.started).Seconds()
	if seconds > 0 {
		p.info.BytesPerSecond = int64(float64(p.info.DownloadedBytes) / seconds)
	}
	if time.Since(p.last) > 200*time.Millisecond {
		setCached(*p.info)
		p.last = time.Now()
	}
	return len(data), nil
}
func maxAssetSize(expected int64) int64 {
	if expected > 0 && expected < 512<<20 {
		return expected
	}
	return 512 << 20
}

func parseSHA256ForAsset(text, assetName string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(strings.TrimPrefix(fields[1], "*"), assetName) {
			return parseSHA256Text(fields[0])
		}
	}
	// A hash-only sidecar is unambiguous; a multi-file manifest is not.
	if len(lines) == 1 && len(strings.Fields(lines[0])) == 1 {
		return parseSHA256Text(lines[0])
	}
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(assetName)) {
			return parseSHA256FromNotes(line, assetName)
		}
	}
	return ""
}
