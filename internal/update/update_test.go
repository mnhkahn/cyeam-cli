package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/version"
)

func TestAssetNameMapsGoArchToReleaseAsset(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"darwin", "arm64", "cyeam_Darwin_arm64.tar.gz"},
		{"darwin", "amd64", "cyeam_Darwin_x86_64.tar.gz"},
		{"linux", "arm64", "cyeam_Linux_arm64.tar.gz"},
		{"linux", "amd64", "cyeam_Linux_x86_64.tar.gz"},
		{"windows", "amd64", "cyeam_Windows_x86_64.zip"},
	}

	for _, tc := range tests {
		t.Run(tc.goos+"_"+tc.goarch, func(t *testing.T) {
			got, err := AssetName(tc.goos, tc.goarch)
			if err != nil {
				t.Fatalf("AssetName: %v", err)
			}
			if got != tc.want {
				t.Fatalf("name = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestArchiveInstallerDownloadsTarGzAndReplacesExecutable(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("new-binary"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	installer := ArchiveInstaller{
		HTTPClient: server.Client(),
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}
	err := installer.Install(context.Background(), Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: server.URL + "/cyeam_Darwin_arm64.tar.gz",
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("exe = %q", got)
	}
}

func makeTarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(data)),
	}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestSelectAssetFindsMatchingReleaseAsset(t *testing.T) {
	asset, err := SelectAsset("mnhkahn/cyeam-cli", "darwin", "arm64", "v1.2.3")
	if err != nil {
		t.Fatalf("SelectAsset: %v", err)
	}
	if asset.BrowserDownloadURL != "https://github.com/mnhkahn/cyeam-cli/releases/download/v1.2.3/cyeam_Darwin_arm64.tar.gz" {
		t.Fatalf("url = %q", asset.BrowserDownloadURL)
	}
}

func TestResultString(t *testing.T) {
	updated := Result{Updated: true, OldVersion: "v0.1.12", NewVersion: "v0.1.13"}
	if updated.String() != "updated: v0.1.12 -> v0.1.13\n" {
		t.Fatalf("updated string = %q", updated.String())
	}

	current := Result{Updated: false, OldVersion: "v0.1.13", NewVersion: "v0.1.13"}
	if current.String() != "already up to date: v0.1.13\n" {
		t.Fatalf("current string = %q", current.String())
	}
}

func TestGitHubUpdaterReturnsCurrentWhenAlreadyLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/tag/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("release page"))
			return
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		http.Redirect(w, r, scheme+"://"+r.Host+"/mnhkahn/cyeam-cli/releases/tag/v0.1.13", http.StatusFound)
	}))
	defer server.Close()

	updater := GitHubUpdater{
		Repo:       "mnhkahn/cyeam-cli",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		GOOS:       "darwin",
		GOARCH:     "arm64",
		Installer:  fakeInstaller{},
	}
	result, err := updater.Update(context.Background(), version.Info{Version: "v0.1.13"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.Updated {
		t.Fatal("expected no update")
	}
	if result.NewVersion != "v0.1.13" {
		t.Fatalf("new version = %q", result.NewVersion)
	}
}

func TestGitHubUpdaterInstallsMatchingAssetWhenNewer(t *testing.T) {
	installer := &recordingInstaller{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/tag/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("release page"))
			return
		}
		secheme := "http"
		if r.TLS != nil {
			secheme = "https"
		}
		http.Redirect(w, r, secheme+"://"+r.Host+"/mnhkahn/cyeam-cli/releases/tag/v0.1.13", http.StatusFound)
	}))
	defer server.Close()

	updater := GitHubUpdater{
		Repo:            "mnhkahn/cyeam-cli",
		BaseURL:         server.URL,
		GOOS:            "darwin",
		GOARCH:          "arm64",
		Installer:       installer,
		DisableChecksum: true, // 这个测试只验证 updater 流程，不测试校验
	}
	result, err := updater.Update(context.Background(), version.Info{Version: "v0.1.12"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !result.Updated {
		t.Fatal("expected update")
	}
	wantURL := "https://github.com/mnhkahn/cyeam-cli/releases/download/v0.1.13/cyeam_Darwin_arm64.tar.gz"
	if installer.asset.BrowserDownloadURL != wantURL {
		t.Fatalf("download url = %q, want %q", installer.asset.BrowserDownloadURL, wantURL)
	}
}

// TestChecksumVerifier 完整测试 checksum 校验流程
func TestChecksumVerifier(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("new-binary"))
	// 计算真实的 sha256 hash
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(archive))
	checksumsTxt := expectedHash + "  cyeam_Darwin_arm64.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(checksumsTxt))
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	// 用装饰器包装真实的 ArchiveInstaller
	baseInstaller := ArchiveInstaller{
		HTTPClient: server.Client(),
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}
	verifier := ChecksumVerifier{
		Next:       baseInstaller,
		HTTPClient: server.Client(),
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: server.URL + "/cyeam_Darwin_arm64.tar.gz",
		ChecksumURL:        server.URL + "/checksums.txt",
	}

	err := verifier.Install(context.Background(), asset)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("exe = %q", got)
	}
}

// TestChecksumVerifierPointerInstaller 测试指针类型的 ArchiveInstaller 也能正常避免重复下载
func TestChecksumVerifierPointerInstaller(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("new-binary"))
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(archive))
	checksumsTxt := expectedHash + "  cyeam_Darwin_arm64.tar.gz\n"

	downloadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(checksumsTxt))
			return
		}
		downloadCount++
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	// 传指针类型的 ArchiveInstaller
	baseInstaller := &ArchiveInstaller{
		HTTPClient: server.Client(),
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}
	verifier := ChecksumVerifier{
		Next:       baseInstaller,
		HTTPClient: server.Client(),
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: server.URL + "/cyeam_Darwin_arm64.tar.gz",
		ChecksumURL:        server.URL + "/checksums.txt",
	}

	err := verifier.Install(context.Background(), asset)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// 应该只下载一次（ChecksumVerifier 下载，PreloadedInstaller 复用 bytes）
	// 实际是 2 次：checksum 下载一次 + archive 下载一次
	if downloadCount != 1 {
		t.Fatalf("expected 1 archive download, got %d (PreloadedInstaller should prevent re-download)", downloadCount)
	}
}

// TestGitHubUpdaterCustomInstallerNotWrapped 验证自定义 Installer 不会被包装 ChecksumVerifier
// 确保向后兼容，不破坏现有自定义 Installer 的使用
func TestGitHubUpdaterCustomInstallerNotWrapped(t *testing.T) {
	// recordingInstaller 是自定义 Installer，不是 ArchiveInstaller
	customInstaller := &recordingInstaller{}

	updater := GitHubUpdater{
		Repo:            "mnhkahn/cyeam-cli",
		GOOS:            "darwin",
		GOARCH:          "arm64",
		Installer:       customInstaller,
		DisableChecksum: false, // 不手动禁用，但自定义 installer 应该不被包装
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: "http://example.com/file.tar.gz",
		ChecksumURL:        "http://example.com/checksums.txt",
	}

	// 应该直接调用自定义 installer，不报错（如果包装了会因为类型不匹配而报错）
	result := Result{NewVersion: "v0.1.13"}
	_, err := updater.installAsset(context.Background(), result, asset)
	if err != nil {
		t.Fatalf("installAsset with custom Installer should succeed without checksum wrapping, got: %v", err)
	}

	// 确认 asset 确实传给了自定义 installer
	if customInstaller.asset.Name != asset.Name {
		t.Fatalf("custom Installer should have received the asset, got: %v", customInstaller.asset)
	}
}

// TestChecksumVerifierReusesNextInstallerHTTPClient 验证 ChecksumVerifier 直接使用时也能正确复用 Next 的自定义 client
// 确保 proxy、auth、custom transport 等配置不会丢失（不需要通过 GitHubUpdater 包装）
func TestChecksumVerifierReusesNextInstallerHTTPClient(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("new-binary"))
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(archive))
	checksumsTxt := expectedHash + "  cyeam_Darwin_arm64.tar.gz\n"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if strings.Contains(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(checksumsTxt))
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	// 直接使用 ChecksumVerifier，不设置 ChecksumVerifier.HTTPClient
	// 只设置 Next installer 的 HTTPClient，应该能正确复用
	baseInstaller := ArchiveInstaller{
		HTTPClient: server.Client(), // 只在 Next 中设置自定义 client
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}

	// ChecksumVerifier.HTTPClient 留空为 nil！
	verifier := ChecksumVerifier{
		Next:       baseInstaller, // 只在 Next 中有 client
		HTTPClient: nil, // 显式不设置
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: server.URL + "/cyeam_Darwin_arm64.tar.gz",
		ChecksumURL:        server.URL + "/checksums.txt",
	}

	// 应该能成功，因为正确复用了 Next installer 的 HTTPClient
	err := verifier.Install(context.Background(), asset)
	if err != nil {
		t.Fatalf("Install should succeed when HTTPClient is taken from Next installer, got: %v", err)
	}
}

// TestChecksumVerifierReusesInstallerHTTPClient 验证 ChecksumVerifier 复用 ArchiveInstaller 配置的自定义 client
// 确保 proxy、auth、custom transport 等配置不会丢失
func TestChecksumVerifierReusesInstallerHTTPClient(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("new-binary"))
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(archive))
	checksumsTxt := expectedHash + "  cyeam_Darwin_arm64.tar.gz\n"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if strings.Contains(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(checksumsTxt))
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	// 使用 server.Client() 作为自定义 client（httptest server 自带特殊 transport）
	baseInstaller := &ArchiveInstaller{
		HTTPClient: server.Client(), // 配置了自定义 client
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}

	updater := NewGitHubUpdater("mnhkahn/cyeam-cli", baseInstaller)
	updater.BaseURL = server.URL
	updater.GOOS = "darwin"
	updater.GOARCH = "arm64"

	// 修改 SelectAsset 返回的 URL 指向测试 server
	asset, _ := SelectAsset("mnhkahn/cyeam-cli", "darwin", "arm64", "v0.1.0")
	asset.BrowserDownloadURL = server.URL + "/cyeam_Darwin_arm64.tar.gz"
	asset.ChecksumURL = server.URL + "/checksums.txt"

	// 通过 ChecksumVerifier 安装，应该能正确复用 server.Client()
	verifier := ChecksumVerifier{Next: baseInstaller}
	err := verifier.Install(context.Background(), asset)
	if err != nil {
		t.Fatalf("Install should succeed with custom HTTP client, got error: %v", err)
	}
}

// TestChecksumVerifierRejectsNonArchiveInstaller 测试 ChecksumVerifier 拒绝非 ArchiveInstaller
// 确保不会出现"校验的 bytes"与"安装的 bytes"不是同一份的安全漏洞
func TestChecksumVerifierRejectsNonArchiveInstaller(t *testing.T) {
	customInstaller := &recordingInstaller{}
	verifier := ChecksumVerifier{
		Next: customInstaller,
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: "http://example.com/file.tar.gz",
		ChecksumURL:        "http://example.com/checksums.txt",
	}

	// 应该明确报错，而不是静默安装未校验的内容
	err := verifier.Install(context.Background(), asset)
	if err == nil {
		t.Fatal("expected error for non-ArchiveInstaller, got nil")
	}
	if !strings.Contains(err.Error(), "ChecksumVerifier requires ArchiveInstaller") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestChecksumVerifierMismatch 测试 hash 不匹配时拒绝安装
func TestChecksumVerifierMismatch(t *testing.T) {
	archive := makeTarGz(t, "cyeam", []byte("malicious-binary"))
	// 用一个错误的 hash
	badChecksums := "0000000000000000000000000000000000000000000000000000000000000000  cyeam_Darwin_arm64.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(badChecksums))
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "cyeam")
	if err := os.WriteFile(exe, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	baseInstaller := ArchiveInstaller{
		HTTPClient: server.Client(),
		ExecutablePath: func() (string, error) {
			return exe, nil
		},
	}
	verifier := ChecksumVerifier{
		Next:       baseInstaller,
		HTTPClient: server.Client(),
	}

	asset := Asset{
		Name:               "cyeam_Darwin_arm64.tar.gz",
		BrowserDownloadURL: server.URL + "/cyeam_Darwin_arm64.tar.gz",
		ChecksumURL:        server.URL + "/checksums.txt",
	}

	err := verifier.Install(context.Background(), asset)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum verification failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证旧文件没有被覆盖
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "old-binary" {
		t.Fatal("binary should not have been replaced on checksum failure")
	}
}

type fakeInstaller struct{}

func (fakeInstaller) Install(ctx context.Context, asset Asset) error {
	return nil
}

type recordingInstaller struct {
	asset Asset
}

func (r *recordingInstaller) Install(ctx context.Context, asset Asset) error {
	r.asset = asset
	return nil
}
