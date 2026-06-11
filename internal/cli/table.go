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

func terminalHyperlink(url string, label string) string {
	return "\033]8;;" + url + "\033\\" + label + "\033]8;;\033\\"
}

func truncateDisplayWidth(s string, maxWidth int) string {
	if displayWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return takeDisplayWidth(s, maxWidth)
	}
	return takeDisplayWidth(s, maxWidth-3) + "..."
}

func takeDisplayWidth(s string, maxWidth int) string {
	var b strings.Builder
	width := 0
	for _, r := range s {
		next := runeWidth(r)
		if width+next > maxWidth {
			break
		}
		b.WriteRune(r)
		width += next
	}
	return b.String()
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	if r == 0 || r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f ||
		r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6))
}
