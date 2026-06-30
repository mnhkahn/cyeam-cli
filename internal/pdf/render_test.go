package pdf

import (
	"strings"
	"testing"
)

func TestRenderMarkdownReportsMissingChineseFont(t *testing.T) {
	originalLoadSystemFonts := loadSystemFonts
	t.Cleanup(func() {
		loadSystemFonts = originalLoadSystemFonts
	})
	loadSystemFonts = func() [][]byte { return nil }

	_, err := RenderMarkdown([]byte("# 中文标题\n\n你好世界"))
	if err == nil {
		t.Fatal("expected missing Chinese font error")
	}
	if !strings.Contains(err.Error(), "no Chinese-capable font") {
		t.Fatalf("expected missing Chinese font error, got %v", err)
	}
}
