package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/version"
)

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ChecksumURL        string `json:"-"` // checksums.txt 下载地址，仅用于本地校验
}

type Result struct {
	Updated    bool
	OldVersion string
	NewVersion string
}

type Installer interface {
	Install(ctx context.Context, asset Asset) error
}

// ChecksumVerifier 装饰器：在安装前校验下载文件的 checksum
// 完全独立于 ArchiveInstaller 实现，通过组合方式增强功能
type ChecksumVerifier struct {
	Next         Installer // 被装饰的实际安装器
	HTTPClient   *http.Client
	ChecksumFunc func(assetName string, archiveBytes []byte, checksums []byte) error // 可注入用于测试
}

func (v ChecksumVerifier) Install(ctx context.Context, asset Asset) error {
	if asset.ChecksumURL == "" {
		return v.Next.Install(ctx, asset)
	}

	// 先做类型检查，避免下载后才发现不支持（浪费带宽 + 不兼容的安全风险）
	switch v.Next.(type) {
	case ArchiveInstaller, *ArchiveInstaller:
		// OK
	default:
		return fmt.Errorf("ChecksumVerifier requires ArchiveInstaller or *ArchiveInstaller, got %T", v.Next)
	}

	// 优先使用 v.HTTPClient，如果没有则尝试从 Next installer 中提取
	// 这样当 ChecksumVerifier 被直接使用时，能正确复用已配置的 custom client
	httpClient := v.HTTPClient
	if httpClient == nil {
		switch ai := v.Next.(type) {
		case ArchiveInstaller:
			httpClient = ai.HTTPClient
		case *ArchiveInstaller:
			httpClient = ai.HTTPClient
		}
		// 如果都没有，使用默认 5 分钟超时 client
		if httpClient == nil {
			httpClient = &http.Client{Timeout: 5 * time.Minute} // 与 ArchiveInstaller 内部超时一致
		}
	}

	// 从 Next installer 中提取 ProgressWriter
	var pw io.Writer = io.Discard
	switch ai := v.Next.(type) {
	case ArchiveInstaller:
		if ai.ProgressWriter != nil {
			pw = ai.ProgressWriter
		}
	case *ArchiveInstaller:
		if ai.ProgressWriter != nil {
			pw = ai.ProgressWriter
		}
	}

	// 下载 checksums.txt
	checksums, err := v.downloadChecksums(ctx, httpClient, asset.ChecksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	// 下载归档文件（带进度条，与 ArchiveInstaller 行为一致
	fmt.Fprintf(pw, "==> Downloading %s\n", asset.Name)
	archiveBytes, err := v.downloadAsset(ctx, httpClient, asset, pw)
	if err != nil {
		return err
	}

	// 执行校验
	checksumFunc := v.ChecksumFunc
	if checksumFunc == nil {
		checksumFunc = VerifyChecksum
	}
	if err := checksumFunc(asset.Name, archiveBytes, checksums); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// 校验通过，使用预加载的 bytes 安装（避免重复下载）
	switch ai := v.Next.(type) {
	case ArchiveInstaller:
		// 输出下载进度，与 ArchiveInstaller.Install 行为一致
		pw := ai.ProgressWriter
		if pw == nil {
			pw = io.Discard
		}
		fmt.Fprintln(pw, "==> Verifying checksum") // 新增：校验进度提示
		return PreloadedInstaller{
			ArchiveBytes:   archiveBytes,
			ProgressWriter: pw,
			ExecutablePath: ai.ExecutablePath,
		}.Install(ctx, asset)
	case *ArchiveInstaller:
		pw := ai.ProgressWriter
		if pw == nil {
			pw = io.Discard
		}
		fmt.Fprintln(pw, "==> Verifying checksum")
		return PreloadedInstaller{
			ArchiveBytes:   archiveBytes,
			ProgressWriter: pw,
			ExecutablePath: ai.ExecutablePath,
		}.Install(ctx, asset)
	default:
		// 理论上不会走到这里，前面已经做过类型检查
		return fmt.Errorf("unexpected installer type: %T", v.Next)
	}
}

// PreloadedInstaller 装饰器：使用预先加载的 bytes，避免重复下载
// 保留与 ArchiveInstaller 一致的进度输出行为
type PreloadedInstaller struct {
	ArchiveBytes   []byte
	ProgressWriter io.Writer
	ExecutablePath func() (string, error)
}

func (pi PreloadedInstaller) Install(ctx context.Context, asset Asset) error {
	pw := pi.ProgressWriter
	if pw == nil {
		pw = io.Discard
	}

	fmt.Fprintln(pw, "==> Extracting archive")
	bin, err := extractBinary(asset.Name, pi.ArchiveBytes)
	if err != nil {
		return err
	}

	fmt.Fprintln(pw, "==> Replacing binary")
	execPath := pi.ExecutablePath
	if execPath == nil {
		execPath = os.Executable
	}
	exe, err := execPath()
	if err != nil {
		return err
	}
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, bin, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (v ChecksumVerifier) downloadChecksums(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (v ChecksumVerifier) downloadAsset(ctx context.Context, client *http.Client, asset Asset, pw io.Writer) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download asset: %s", resp.Status)
	}
	body := newProgressReader(resp.Body, resp.ContentLength, pw)
	return io.ReadAll(body)
}

// VerifyChecksum 校验归档文件的 checksum
func VerifyChecksum(assetName string, archiveBytes []byte, checksumsContent []byte) error {
	expected := parseChecksum(checksumsContent, assetName)
	if expected == "" {
		return fmt.Errorf("no checksum found for %s", assetName)
	}

	actual := fmt.Sprintf("%x", sha256.Sum256(archiveBytes))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return fmt.Errorf("hash mismatch (expected %s, got %s)", expected, actual)
	}
	return nil
}

// parseChecksum 从 checksums.txt 中提取指定文件名的 hash
func parseChecksum(checksumsContent []byte, filename string) string {
	scanner := bufio.NewScanner(bytes.NewReader(checksumsContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 格式: "sha256-hash  filename"
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			return parts[0]
		}
	}
	return ""
}

type ArchiveInstaller struct {
	HTTPClient     *http.Client
	ExecutablePath func() (string, error)
	ProgressWriter io.Writer // optional: writes status and download progress; use os.Stderr for CLI
}

// progressReader counts bytes read and prints download progress hashes to ProgressWriter.
type progressReader struct {
	r     io.Reader
	pw    io.Writer
	total int64
	cur   int64
	last  int // last printed percentage (0-100)
}

func newProgressReader(r io.Reader, total int64, pw io.Writer) io.Reader {
	if pw == nil {
		return r
	}
	return &progressReader{r: r, pw: pw, total: total, last: -1}
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.cur += int64(n)
	if pr.total > 0 {
		pct := int(float64(pr.cur*100)/float64(pr.total) + 0.5)
		if pct > pr.last {
			pr.last = pct
			hashes := pct * 70 / 100
			fmt.Fprintf(pr.pw, "\r%s%3d%%", strings.Repeat("#", hashes), pct)
			if pct == 100 {
				fmt.Fprintln(pr.pw)
			}
		}
	}
	return
}

func (i ArchiveInstaller) Install(ctx context.Context, asset Asset) error {
	httpClient := i.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	pw := i.ProgressWriter
	if pw == nil {
		pw = io.Discard
	}
	executablePath := i.ExecutablePath
	if executablePath == nil {
		executablePath = os.Executable
	}

	fmt.Fprintln(pw, "==> Downloading release binary")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(pw, "==> Downloading %s\n", asset.Name)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	body := newProgressReader(resp.Body, resp.ContentLength, pw)
	archiveBytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	fmt.Fprintln(pw, "==> Extracting archive")
	bin, err := extractBinary(asset.Name, archiveBytes)
	if err != nil {
		return err
	}
	fmt.Fprintln(pw, "==> Replacing binary")
	exe, err := executablePath()
	if err != nil {
		return err
	}
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, bin, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

type GitHubUpdater struct {
	Repo             string
	BaseURL          string
	HTTPClient       *http.Client
	GOOS             string
	GOARCH           string
	Installer        Installer
	DisableChecksum  bool // 可选：禁用 checksum 校验（不推荐）
}

func extractBinary(assetName string, archiveBytes []byte) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractZipBinary(archiveBytes)
	}
	return extractTarGzBinary(archiveBytes)
}

func extractTarGzBinary(archiveBytes []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archiveBytes))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if isCyeamBinary(header.Name) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("cyeam binary not found in tar.gz")
}

func extractZipBinary(archiveBytes []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return nil, err
	}
	for _, file := range zr.File {
		if !isCyeamBinary(file.Name) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		return data, nil
	}
	return nil, fmt.Errorf("cyeam binary not found in zip")
}

func isCyeamBinary(name string) bool {
	base := filepath.Base(name)
	return base == "cyeam" || base == "cyeam.exe"
}

func NewGitHubUpdater(repo string, installer Installer) *GitHubUpdater {
	return &GitHubUpdater{
		Repo:       repo,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Installer:  installer,
	}
}

func (u GitHubUpdater) Update(ctx context.Context, current version.Info) (Result, error) {
	tag, err := u.latestTag(ctx)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Updated:    false,
		OldVersion: current.Version,
		NewVersion: tag,
	}
	if current.Version == tag {
		return result, nil
	}
	asset, err := SelectAsset(u.Repo, u.GOOS, u.GOARCH, tag)
	if err != nil {
		return Result{}, err
	}
	return u.installAsset(ctx, result, asset)
}

func (u GitHubUpdater) InstallURL(ctx context.Context, downloadURL string) error {
	if u.Installer == nil {
		return fmt.Errorf("installer is required")
	}
	return u.Installer.Install(ctx, Asset{BrowserDownloadURL: downloadURL})
}

func (u GitHubUpdater) installAsset(ctx context.Context, result Result, asset Asset) (Result, error) {
	if u.Installer == nil {
		return Result{}, fmt.Errorf("installer is required")
	}

	installer := u.Installer
	// 只对 ArchiveInstaller 自动应用 checksum 校验
	// 其他自定义 Installer 可能有自己的下载逻辑，无法保证校验的 bytes == 安装的 bytes
	if !u.DisableChecksum {
		switch ai := installer.(type) {
		case ArchiveInstaller:
			// 复用已配置的 HTTPClient，只有 nil 时才用默认的 5 分钟超时
			downloadClient := ai.HTTPClient
			if downloadClient == nil {
				downloadClient = &http.Client{Timeout: 5 * time.Minute}
			}
			installer = ChecksumVerifier{
				Next:       ai,
				HTTPClient: downloadClient,
			}
		case *ArchiveInstaller:
			downloadClient := ai.HTTPClient
			if downloadClient == nil {
				downloadClient = &http.Client{Timeout: 5 * time.Minute}
			}
			installer = ChecksumVerifier{
				Next:       ai,
				HTTPClient: downloadClient,
			}
		}
		// 其他类型 installer：不应用校验，直接使用原 installer
	}

	if err := installer.Install(ctx, asset); err != nil {
		return Result{}, err
	}
	result.Updated = true
	return result, nil
}

func (u GitHubUpdater) latestTag(ctx context.Context) (string, error) {
	base := strings.TrimRight(u.BaseURL, "/")
	if base == "" {
		base = "https://github.com"
	}
	url := base + "/" + u.Repo + "/releases/latest"
	httpClient := u.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html, */*")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github release page: %s", resp.Status)
	}
	finalURL := resp.Request.URL.String()
	tag := extractTagFromURL(finalURL)
	if tag == "" {
		return "", fmt.Errorf("could not extract version tag from %s", finalURL)
	}
	return tag, nil
}

func extractTagFromURL(rawURL string) string {
	// e.g. https://github.com/mnhkahn/cyeam-cli/releases/tag/v0.1.4
	idx := strings.Index(rawURL, "/releases/tag/")
	if idx < 0 {
		return ""
	}
	tag := rawURL[idx+len("/releases/tag/"):]
	tag = strings.TrimRight(tag, "/")
	return tag
}

func (r Result) String() string {
	if !r.Updated {
		return fmt.Sprintf("already up to date: %s\n", r.NewVersion)
	}
	return fmt.Sprintf("updated: %s -> %s\n", r.OldVersion, r.NewVersion)
}

func AssetName(goos string, goarch string) (string, error) {
	osName, err := releaseOS(goos)
	if err != nil {
		return "", err
	}
	archName, err := releaseArch(goarch)
	if err != nil {
		return "", err
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("cyeam_%s_%s%s", osName, archName, ext), nil
}

func SelectAsset(repo, goos, goarch, tag string) (Asset, error) {
	name, err := AssetName(goos, goarch)
	if err != nil {
		return Asset{}, err
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, name)
	checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", repo, tag)
	return Asset{Name: name, BrowserDownloadURL: url, ChecksumURL: checksumURL}, nil
}

func releaseOS(goos string) (string, error) {
	switch goos {
	case "darwin":
		return "Darwin", nil
	case "linux":
		return "Linux", nil
	case "windows":
		return "Windows", nil
	default:
		return "", fmt.Errorf("unsupported OS %q", goos)
	}
}

func releaseArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture %q", goarch)
	}
}
