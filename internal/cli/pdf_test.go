package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/mnhkahn/cyeam-cli/internal/pdf"
	"io"
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

func TestPDFArithmeticMode(t *testing.T) {
	original := renderArithmeticPDF
	t.Cleanup(func() { renderArithmeticPDF = original })
	renderArithmeticPDF = func(src []byte, opts pdf.ArithmeticOptions) ([]byte, error) {
		if string(src) != "3×5＝\n" || opts.Layout != "quantity" || opts.Title != "练习" {
			t.Fatalf("unexpected input %q %+v", src, opts)
		}
		return []byte("%PDF arithmetic"), nil
	}
	cmd := newPDFCommand()
	cmd.SetIn(strings.NewReader("3×5＝\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--mode", "arithmetic", "--layout", "quantity", "--title", "练习"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		PDF string `json:"pdf"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(result.PDF)
	if err != nil || string(decoded) != "%PDF arithmetic" {
		t.Fatalf("unexpected output %s: %v", out.String(), err)
	}
}

func TestPDFRejectsInvalidModeFlags(t *testing.T) {
	for _, args := range [][]string{{"--mode", "invalid"}, {"--layout", "grade4"}, {"--title", "test"}} {
		cmd := newPDFCommand()
		cmd.SetArgs(args)
		cmd.SetIn(strings.NewReader("1+1="))
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}

func TestPDFPinyinMode(t *testing.T) {
	original := renderPinyinPDF
	t.Cleanup(func() { renderPinyinPDF = original })
	renderPinyinPDF = func(src []byte) ([]byte, error) {
		if string(src) != "你好 世界" {
			t.Fatalf("unexpected input %q", src)
		}
		return []byte("%PDF pinyin"), nil
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "words.txt")
	output := filepath.Join(dir, "words.pdf")
	if err := os.WriteFile(input, []byte("你好 世界"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, fileInput := range []bool{false, true} {
		cmd := newPDFCommand()
		args := []string{"--mode", "pinyin", "-o", output}
		if fileInput {
			args = append(args, input)
		} else {
			cmd.SetIn(strings.NewReader("你好 世界"))
		}
		cmd.SetArgs(args)
		cmd.SetOut(io.Discard)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil || string(data) != "%PDF pinyin" {
			t.Fatalf("output %q, error %v", data, err)
		}
	}
}
