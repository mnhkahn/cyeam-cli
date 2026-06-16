package releaseworkflow

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseWorkflowPublishesReleaseAndNotifiesFeishu(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	text := string(body)

	assertContains(t, text, "tags:")
	assertContains(t, text, "v*")
	assertContains(t, text, "contents: write")
	assertContains(t, text, "goreleaser/goreleaser-action")
	assertContains(t, text, "GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}")
	assertContains(t, text, "RELEASE_WEBHOOK_URL")
	assertContains(t, text, "changelog")
	assertContains(t, text, "continue-on-error: true")
}

func TestGoReleaserBuildsExpectedArchives(t *testing.T) {
	body, err := os.ReadFile("../../.goreleaser.yaml")
	if err != nil {
		t.Fatalf("read goreleaser config: %v", err)
	}
	text := string(body)

	assertContains(t, text, "main: ./cmd/cyeam")
	assertContains(t, text, "binary: cyeam")
	assertContains(t, text, "darwin")
	assertContains(t, text, "linux")
	assertContains(t, text, "windows")
	assertContains(t, text, "arm64")
	assertContains(t, text, "amd64")
	assertContains(t, text, "name_template: \"cyeam_{{ title .Os }}_{{ if eq .Arch \\\"amd64\\\" }}x86_64{{ else }}{{ .Arch }}{{ end }}\"")
	assertContains(t, text, "name_template: checksums.txt")
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in:\n%s", want, text)
	}
}