# CLI Table Renderer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ad hoc list output with a shared screenshot-style terminal table renderer for `roadbook list` and `cnote list`.

**Architecture:** Move table rendering out of `internal/cli/root.go` into a focused `internal/cli/table.go`. The renderer accepts headers and rows as `tableCell` values, calculates display width from visible text, and renders Unicode borders plus purple header text. Domain commands build rows and call small list-specific helpers.

**Tech Stack:** Go 1.23, Cobra command handlers, existing package-level unit tests in `internal/cli`.

---

## File Structure

- Create `internal/cli/table.go`: shared Unicode table renderer, ANSI constants, width/truncation helpers, terminal hyperlink helper.
- Create `internal/cli/table_test.go`: focused renderer tests for borders, purple headers, wide characters, and hyperlinks.
- Modify `internal/cli/root.go`: remove embedded table renderer helpers, keep roadbook row mapping, add cnote row mapping, call shared helpers.
- Modify `internal/cli/root_test.go`: update roadbook table expectations and add cnote table rendering expectations.

## Task 1: Add Shared Renderer Tests

**Files:**
- Create: `internal/cli/table_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli -run 'TestRenderTableUsesUnicodeBordersAndPurpleHeaders|TestRenderTableUsesVisibleWidthForHyperlinks|TestTruncateDisplayWidthHandlesWideCharacters'`

Expected: fail because `renderTable` and `cliTable` are not defined yet.

## Task 2: Implement Shared Renderer

**Files:**
- Create: `internal/cli/table.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Move table primitives into `internal/cli/table.go`**

```go
package cli

import (
	"fmt"
	"io"
	"strings"
)

const (
	tableHeaderColor = "\033[35m"
	tableResetColor  = "\033[0m"
)

type cliTable struct {
	Headers []tableCell
	Rows    [][]tableCell
	Color   bool
}

type tableCell struct {
	text    string
	visible string
}
```

- [ ] **Step 2: Implement Unicode border rendering**

```go
func renderTable(out io.Writer, t cliTable) error {
	widths := tableColumnWidths(t)
	if _, err := fmt.Fprintln(out, tableBorder(widths, "┌", "┬", "┐")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, tableRow(colorHeaderCells(t.Headers, t.Color), widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, tableBorder(widths, "├", "┼", "┤")); err != nil {
		return err
	}
	for _, row := range t.Rows {
		if _, err := fmt.Fprintln(out, tableRow(row, widths)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, tableBorder(widths, "└", "┴", "┘"))
	return err
}
```

- [ ] **Step 3: Implement width, row, border, color, hyperlink, truncation helpers**

```go
func tableColumnWidths(t cliTable) []int {
	widths := make([]int, len(t.Headers))
	for i, cell := range t.Headers {
		widths[i] = max(widths[i], displayWidth(cell.visible))
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			widths[i] = max(widths[i], displayWidth(cell.visible))
		}
	}
	return widths
}

func colorHeaderCells(cells []tableCell, color bool) []tableCell {
	if !color {
		return cells
	}
	out := make([]tableCell, len(cells))
	for i, cell := range cells {
		out[i] = tableCell{
			text:    tableHeaderColor + cell.text + tableResetColor,
			visible: cell.visible,
		}
	}
	return out
}

func tableBorder(widths []int, left string, join string, right string) string {
	var b strings.Builder
	b.WriteString(left)
	for i, width := range widths {
		if i > 0 {
			b.WriteString(join)
		}
		b.WriteString(strings.Repeat("─", width+2))
	}
	b.WriteString(right)
	return b.String()
}

func tableRow(row []tableCell, widths []int) string {
	var b strings.Builder
	b.WriteString("│")
	for i, width := range widths {
		cell := tableCell{}
		if i < len(row) {
			cell = row[i]
		}
		b.WriteByte(' ')
		b.WriteString(cell.text)
		b.WriteString(strings.Repeat(" ", width-displayWidth(cell.visible)))
		b.WriteByte(' ')
		b.WriteString("│")
	}
	return b.String()
}
```

- [ ] **Step 4: Remove old ASCII table helpers from `root.go`**

Remove `tableWidths`, old `tableBorder(widths []int)`, and the old `tableRow` implementation from `internal/cli/root.go`. Keep or move `terminalHyperlink`, `truncateDisplayWidth`, `takeDisplayWidth`, `displayWidth`, `runeWidth`, and `isWideRune` in `table.go`.

- [ ] **Step 5: Run renderer tests**

Run: `go test ./internal/cli -run 'TestRenderTableUsesUnicodeBordersAndPurpleHeaders|TestRenderTableUsesVisibleWidthForHyperlinks|TestTruncateDisplayWidthHandlesWideCharacters'`

Expected: pass.

## Task 3: Wire Roadbook And CNote Lists

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

- [ ] **Step 1: Update roadbook helper to call `renderTable`**

```go
func renderRoadbookListTable(out io.Writer, rows []roadbookListRow) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "标题", visible: "标题"},
			{text: "修改时间", visible: "修改时间"},
			{text: "链接", visible: "链接"},
		},
		Color: true,
	}
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = "-"
		}
		title = truncateDisplayWidth(title, 32)
		link := terminalHyperlink(roadbookURL(row.Name), "链接")
		t.Rows = append(t.Rows, []tableCell{
			{text: title, visible: title},
			{text: row.Modified, visible: row.Modified},
			{text: link, visible: "链接"},
		})
	}
	return renderTable(out, t)
}
```

- [ ] **Step 2: Add cnote row type and render helper**

```go
type cnoteListRow struct {
	Name     string
	Modified string
}

func renderCnoteListTable(out io.Writer, rows []cnoteListRow) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "文件名", visible: "文件名"},
			{text: "修改时间", visible: "修改时间"},
		},
		Color: true,
	}
	for _, row := range rows {
		name := truncateDisplayWidth(row.Name, 32)
		t.Rows = append(t.Rows, []tableCell{
			{text: name, visible: name},
			{text: row.Modified, visible: row.Modified},
		})
	}
	return renderTable(out, t)
}
```

- [ ] **Step 3: Update `newCnoteListCommand` to build rows**

```go
var rows []cnoteListRow
for _, item := range items {
	rows = append(rows, cnoteListRow{
		Name:     strings.TrimSuffix(item.Name, ".html"),
		Modified: item.LastModifiedDateTime[:10],
	})
}
return renderCnoteListTable(deps.Stdout, rows)
```

- [ ] **Step 4: Update roadbook test expectations**

In `TestRenderRoadbookListTableUsesHyperlinkLabel`, expect `┌`, `─`, `│`, purple header `\033[35m标题\033[0m`, and no `+` ASCII border.

- [ ] **Step 5: Add cnote render helper test**

```go
func TestRenderCnoteListTableUsesSharedTableStyle(t *testing.T) {
	stdout := new(bytes.Buffer)
	rows := []cnoteListRow{{
		Name:     "日记",
		Modified: "2026-06-12",
	}}

	if err := renderCnoteListTable(stdout, rows); err != nil {
		t.Fatalf("render cnote table: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"┌", "├", "└", "│", "\033[35m文件名\033[0m", "\033[35m修改时间\033[0m", "日记", "2026-06-12"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 6: Run CLI package tests**

Run: `go test ./internal/cli`

Expected: pass.

## Task 4: Full Verification And Commit

**Files:**
- Verify all changed files.

- [ ] **Step 1: Format Go files**

Run: `gofmt -w internal/cli/root.go internal/cli/root_test.go internal/cli/table.go internal/cli/table_test.go`

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`

Expected: all packages pass.

- [ ] **Step 3: Check diff**

Run: `git diff --check`

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add -- internal/cli/root.go internal/cli/root_test.go internal/cli/table.go internal/cli/table_test.go docs/superpowers/plans/2026-06-12-cli-table-renderer-implementation.md
git commit -m "feat(cli): 统一列表表格输出样式"
```
