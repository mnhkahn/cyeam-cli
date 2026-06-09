package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
			{Name: "cyeam_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/cyeam_Darwin_arm64.tar.gz"},
		},
	}

	asset, err := SelectAsset(release, "darwin", "arm64")
	if err != nil {
		t.Fatalf("SelectAsset: %v", err)
	}
	if asset.BrowserDownloadURL != "https://example.com/cyeam_Darwin_arm64.tar.gz" {
		t.Fatalf("url = %q", asset.BrowserDownloadURL)
	}
}

func TestResultString(t *testing.T) {
	updated := Result{Updated: true, OldVersion: "v1.0.0", NewVersion: "v1.1.0"}
	if updated.String() != "updated: v1.0.0 -> v1.1.0\n" {
		t.Fatalf("updated string = %q", updated.String())
	}

	current := Result{Updated: false, OldVersion: "v1.1.0", NewVersion: "v1.1.0"}
	if current.String() != "already up to date: v1.1.0\n" {
		t.Fatalf("current string = %q", current.String())
	}
}

func TestGitHubUpdaterReturnsCurrentWhenAlreadyLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/mnhkahn/cyeam-cli/releases/latest" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.1.0","assets":[]}`))
	}))
	defer server.Close()

	updater := GitHubUpdater{
		Repo:       "mnhkahn/cyeam-cli",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
		GOOS:       "darwin",
		GOARCH:     "arm64",
		Installer:  fakeInstaller{},
	}
	result, err := updater.Update(context.Background(), version.Info{Version: "v1.1.0"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.Updated {
		t.Fatal("expected no update")
	}
	if result.NewVersion != "v1.1.0" {
		t.Fatalf("new version = %q", result.NewVersion)
	}
}

func TestGitHubUpdaterInstallsMatchingAssetWhenNewer(t *testing.T) {
	installer := &recordingInstaller{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name":"v1.1.0",
			"assets":[
				{"name":"cyeam_Darwin_arm64.tar.gz","browser_download_url":"https://example.com/cyeam_Darwin_arm64.tar.gz"}
			]
		}`))
	}))
	defer server.Close()

	updater := GitHubUpdater{
		Repo:       "mnhkahn/cyeam-cli",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
		GOOS:       "darwin",
		GOARCH:     "arm64",
		Installer:  installer,
	}
	result, err := updater.Update(context.Background(), version.Info{Version: "v1.0.0"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !result.Updated {
		t.Fatal("expected update")
	}
	if installer.asset.Name != "cyeam_Darwin_arm64.tar.gz" {
		t.Fatalf("asset = %q", installer.asset.Name)
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
