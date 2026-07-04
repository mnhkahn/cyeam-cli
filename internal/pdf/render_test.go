package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMarkdownColumnsSyntaxRendersWithoutMarkers(t *testing.T) {
	pdfData, err := RenderMarkdown([]byte("::: columns 2\n::: column\n左列\n:::\n::: column\n右列\n:::\n:::\n"))
	if err != nil {
		t.Fatalf("RenderMarkdown columns returned error: %v", err)
	}
	if bytes.Contains(pdfData, []byte("columns 2")) || bytes.Contains(pdfData, []byte(":::")) {
		t.Fatalf("expected columns markers to be layout syntax, got raw markers in PDF bytes")
	}
}

func TestSplitMarkdownLayoutExtractsColumns(t *testing.T) {
	segments := splitMarkdownLayout([]byte("# 标题\n\n::: columns 2\n::: column\n左列\n:::\n::: column\n右列\n:::\n:::\n\n结尾"))
	if len(segments) != 3 {
		t.Fatalf("expected normal, columns, normal segments, got %d", len(segments))
	}
	if string(segments[0].src) != "# 标题\n\n" {
		t.Fatalf("first segment = %q", segments[0].src)
	}
	if len(segments[1].columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(segments[1].columns))
	}
	if string(segments[1].columns[0]) != "左列\n" || string(segments[1].columns[1]) != "右列\n" {
		t.Fatalf("columns = %#v", segments[1].columns)
	}
	if string(segments[2].src) != "\n结尾" {
		t.Fatalf("last segment = %q", segments[2].src)
	}
}

func TestRenderTypstUsesCompiler(t *testing.T) {
	originalCompileTypst := compileTypst
	t.Cleanup(func() {
		compileTypst = originalCompileTypst
	})

	var gotSrc []byte
	compileTypst = func(src []byte) ([]byte, error) {
		gotSrc = append([]byte(nil), src...)
		return []byte("%PDF typst"), nil
	}

	pdfData, err := RenderTypst([]byte("#columns(2)[A #colbreak() B]"))
	if err != nil {
		t.Fatalf("RenderTypst returned error: %v", err)
	}
	if string(pdfData) != "%PDF typst" {
		t.Fatalf("RenderTypst returned %q", pdfData)
	}
	if string(gotSrc) != "#columns(2)[A #colbreak() B]" {
		t.Fatalf("RenderTypst passed source %q", gotSrc)
	}
}

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

func TestSelectPDFFontSkipsChineseFontThatGofpdfCannotLoad(t *testing.T) {
	badFont := []byte("bad")
	goodFont := []byte("good")
	restore := stubFontSelection(t, [][]byte{badFont, goodFont}, func(fontBytes []byte, text string) bool {
		return true
	}, func(fontBytes []byte) error {
		if bytes.Equal(fontBytes, badFont) {
			return errors.New("gofpdf rejected font")
		}
		return nil
	})
	defer restore()

	got, err := selectPDFFont([]byte("中文"))
	if err != nil {
		t.Fatalf("expected usable fallback font, got error: %v", err)
	}
	if !bytes.Equal(got, goodFont) {
		t.Fatalf("expected second candidate, got %q", got)
	}
}

func TestSelectPDFFontReportsAllChineseFontsFailedToLoad(t *testing.T) {
	restore := stubFontSelection(t, [][]byte{[]byte("bad-1"), []byte("bad-2")}, func(fontBytes []byte, text string) bool {
		return true
	}, func(fontBytes []byte) error {
		return errors.New("gofpdf rejected font")
	})
	defer restore()

	_, err := selectPDFFont([]byte("中文"))
	if err == nil {
		t.Fatal("expected all fonts failed error")
	}
	if !strings.Contains(err.Error(), "Chinese-capable fonts found but failed to load") {
		t.Fatalf("expected load failure error, got %v", err)
	}
	if strings.Contains(err.Error(), "no Chinese-capable font") {
		t.Fatalf("expected load failure to be distinguished from missing font, got %v", err)
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

func stubFontSelection(t *testing.T, fonts [][]byte, supports func([]byte, string) bool, validate func([]byte) error) func() {
	t.Helper()
	originalLoadSystemFonts := loadSystemFonts
	originalFontSupportsHan := fontSupportsHan
	originalValidatePDFFont := validatePDFFont
	loadSystemFonts = func() [][]byte { return fonts }
	fontSupportsHan = supports
	validatePDFFont = validate
	return func() {
		loadSystemFonts = originalLoadSystemFonts
		fontSupportsHan = originalFontSupportsHan
		validatePDFFont = originalValidatePDFFont
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
