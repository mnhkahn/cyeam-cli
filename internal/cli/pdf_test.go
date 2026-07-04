package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFCommandRendersTypstFiles(t *testing.T) {
	originalRenderTypstPDF := renderTypstPDF
	t.Cleanup(func() {
		renderTypstPDF = originalRenderTypstPDF
	})

	var gotSrc []byte
	renderTypstPDF = func(src []byte) ([]byte, error) {
		gotSrc = append([]byte(nil), src...)
		return []byte("%PDF typst"), nil
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "layout.typ")
	outPath := filepath.Join(dir, "layout.pdf")
	if err := os.WriteFile(srcPath, []byte("#columns(2)[A #colbreak() B]"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newPDFCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{srcPath, "-o", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pdf command returned error: %v", err)
	}
	if string(gotSrc) != "#columns(2)[A #colbreak() B]" {
		t.Fatalf("renderTypstPDF got source %q", gotSrc)
	}
	pdfData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(pdfData) != "%PDF typst" {
		t.Fatalf("saved PDF = %q", pdfData)
	}
	if !strings.Contains(out.String(), "saved: "+outPath) {
		t.Fatalf("stdout = %q", out.String())
	}
}
