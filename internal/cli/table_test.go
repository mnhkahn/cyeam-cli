package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTableUsesUnicodeBordersAndPurpleHeaders(t *testing.T) {
	out := new(bytes.Buffer)
	err := renderTable(out, cliTable{
		Headers: []tableCell{
			{text: "标题", visible: "标题"},
			{text: "修改时间", visible: "修改时间"},
		},
		Rows: [][]tableCell{{
			{text: "成都三日游", visible: "成都三日游"},
			{text: "2026-06-10", visible: "2026-06-10"},
		}},
		Color: true,
	})
	if err != nil {
		t.Fatalf("render table: %v", err)
	}

	got := out.String()
	for _, want := range []string{"┌", "┬", "┐", "├", "┼", "┤", "└", "┴", "┘", "│"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table missing border %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "\033[35m标题\033[0m") {
		t.Fatalf("header is not purple:\n%s", got)
	}
	if strings.Contains(got, "+") || strings.Contains(got, "|") {
		t.Fatalf("table still uses ASCII borders:\n%s", got)
	}
}

func TestRenderTableUsesVisibleWidthForHyperlinks(t *testing.T) {
	out := new(bytes.Buffer)
	url := "https://www.cyeam.com/tool/roadbook?id=p_abc123"
	link := terminalHyperlink(url, "链接")

	err := renderTable(out, cliTable{
		Headers: []tableCell{{text: "链接", visible: "链接"}},
		Rows:    [][]tableCell{{{text: link, visible: "链接"}}},
		Color:   true,
	})
	if err != nil {
		t.Fatalf("render table: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, link) {
		t.Fatalf("table missing hyperlink label:\n%s", got)
	}
	visible := strings.ReplaceAll(got, link, "链接")
	if strings.Contains(visible, url) {
		t.Fatalf("table exposes raw URL outside hyperlink escape:\n%s", got)
	}
	if !strings.Contains(visible, "│ \033[35m链接\033[0m │") {
		t.Fatalf("header width does not use visible text:\n%s", got)
	}
}

func TestTruncateDisplayWidthHandlesWideCharacters(t *testing.T) {
	got := truncateDisplayWidth("成都三日游路线", 8)
	if got != "成都..." {
		t.Fatalf("truncated = %q, want %q", got, "成都...")
	}
}
