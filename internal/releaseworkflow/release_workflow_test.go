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
	assertContains(t, text, "ref: ${{ github.sha }}")
	assertContains(t, text, "Verify release tag")
	assertContains(t, text, "GORELEASER_CURRENT_TAG: ${{ github.ref_name }}")
	assertContains(t, text, "goreleaser/goreleaser-action")
	assertContains(t, text, "GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}")
	assertContains(t, text, "FEISHU_WEBHOOK_URL")
	assertContains(t, text, "git-chglog --template .chglog/webhook.tpl.md")
	assertContains(t, text, "continue-on-error: true")
	for _, command := range []string{"ref: main", "git push", "git tag -f", "gh release delete"} {
		if strings.Contains(text, command) {
			t.Errorf("release workflow must not contain %q", command)
		}
	}
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
	assertContains(t, text, "changelog:")
	assertContains(t, text, "replace_existing_artifacts: false")
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in:\n%s", want, text)
	}
}
