package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
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

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Result struct {
	Updated    bool
	OldVersion string
	NewVersion string
}

type Installer interface {
	Install(ctx context.Context, asset Asset) error
}

type ArchiveInstaller struct {
	HTTPClient     *http.Client
	ExecutablePath func() (string, error)
}

func (i ArchiveInstaller) Install(ctx context.Context, asset Asset) error {
	httpClient := i.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	executablePath := i.ExecutablePath
	if executablePath == nil {
		executablePath = os.Executable
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	archiveBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bin, err := extractBinary(asset.Name, archiveBytes)
	if err != nil {
		return err
	}
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
	Repo       string
	APIBaseURL string
	HTTPClient *http.Client
	GOOS       string
	GOARCH     string
	Installer  Installer
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
		APIBaseURL: "https://api.github.com",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Installer:  installer,
	}
}

func (u GitHubUpdater) Update(ctx context.Context, current version.Info) (Result, error) {
	release, err := u.latestRelease(ctx)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Updated:    false,
		OldVersion: current.Version,
		NewVersion: release.TagName,
	}
	if current.Version == release.TagName {
		return result, nil
	}
	asset, err := SelectAsset(release, u.GOOS, u.GOARCH)
	if err != nil {
		return Result{}, err
	}
	if u.Installer == nil {
		return Result{}, fmt.Errorf("installer is required")
	}
	if err := u.Installer.Install(ctx, asset); err != nil {
		return Result{}, err
	}
	result.Updated = true
	return result, nil
}

func (u GitHubUpdater) latestRelease(ctx context.Context) (Release, error) {
	apiBase := strings.TrimRight(u.APIBaseURL, "/")
	httpClient := u.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/repos/"+u.Repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Release{}, fmt.Errorf("github release request failed: %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, err
	}
	if release.TagName == "" {
		return Release{}, fmt.Errorf("github release has empty tag")
	}
	return release, nil
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

func SelectAsset(release Release, goos string, goarch string) (Asset, error) {
	want, err := AssetName(goos, goarch)
	if err != nil {
		return Asset{}, err
	}
	for _, asset := range release.Assets {
		if asset.Name == want {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("release asset %q not found in %s", want, release.TagName)
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
