package pdf

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSystemFontsReadsTTFAndTTCOnly(t *testing.T) {
	dir := t.TempDir()
	ttfPath := filepath.Join(dir, "NotoSansCJK.ttf")
	otfPath := filepath.Join(dir, "skip.otf")
	ttcPath := filepath.Join(dir, "NotoSansCJK.ttc")
	if err := os.WriteFile(ttfPath, pdfFont, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otfPath, []byte("otf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ttcPath, makeSingleFontTTC(t, pdfFont), 0o644); err != nil {
		t.Fatal(err)
	}

	originalFindFontPath := findFontPath
	originalListFontPaths := listFontPaths
	t.Cleanup(func() {
		findFontPath = originalFindFontPath
		listFontPaths = originalListFontPaths
	})
	findFontPath = func(string) (string, error) { return otfPath, nil }
	listFontPaths = func() []string {
		return []string{otfPath, ttcPath, ttfPath}
	}

	got := loadSystemFonts()
	if len(got) != 2 {
		t.Fatalf("expected TTF and extracted TTC font, got %d candidates", len(got))
	}
	for _, fontBytes := range got {
		if _, err := newRendererWithFont(fontBytes); err != nil {
			t.Fatalf("expected loadable TrueType candidate: %v", err)
		}
	}
}

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

func TestNewRendererWithFontReportsGofpdfFontLoadFailure(t *testing.T) {
	_, err := newRendererWithFont([]byte("not a truetype font"))
	if err == nil {
		t.Fatal("expected font load error")
	}
	if !strings.Contains(err.Error(), "load PDF font") {
		t.Fatalf("expected font load error, got %v", err)
	}
}

func makeSingleFontTTC(t *testing.T, ttf []byte) []byte {
	t.Helper()
	if len(ttf) < 12 {
		t.Fatal("test TTF too short")
	}
	numTables := int(binary.BigEndian.Uint16(ttf[4:6]))
	dirLen := 12 + numTables*16
	if len(ttf) < dirLen {
		t.Fatal("test TTF table directory too short")
	}

	fontOffset := 16
	ttc := make([]byte, fontOffset+len(ttf))
	copy(ttc[0:4], []byte("ttcf"))
	binary.BigEndian.PutUint32(ttc[4:8], 0x00010000)
	binary.BigEndian.PutUint32(ttc[8:12], 1)
	binary.BigEndian.PutUint32(ttc[12:16], uint32(fontOffset))
	copy(ttc[fontOffset:], ttf)

	for i := 0; i < numTables; i++ {
		rec := fontOffset + 12 + i*16
		oldOffset := binary.BigEndian.Uint32(ttc[rec+8 : rec+12])
		binary.BigEndian.PutUint32(ttc[rec+8:rec+12], oldOffset+uint32(fontOffset))
	}
	return ttc
}
