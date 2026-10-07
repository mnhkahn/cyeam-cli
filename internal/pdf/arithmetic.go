package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// ArithmeticOptions selects a worksheet layout, not a question generator.
type ArithmeticOptions struct {
	Layout string
	Title  string
}

type arithmeticLayout struct {
	rows, cols   int
	height, size float64
}

func arithmeticLayoutFor(name string) (arithmeticLayout, error) {
	switch name {
	case "", "standard":
		return arithmeticLayout{20, 5, 12, 11.5}, nil
	case "grade4":
		return arithmeticLayout{25, 4, 10, 9}, nil
	case "large-number", "negative":
		return arithmeticLayout{25, 3, 10, 10}, nil
	case "quantity":
		return arithmeticLayout{25, 2, 10, 10}, nil
	case "vertical":
		return arithmeticLayout{5, 5, 48, 11.5}, nil
	default:
		return arithmeticLayout{}, fmt.Errorf("unknown arithmetic layout %q (use standard, grade4, large-number, negative, quantity, vertical)", name)
	}
}

// RenderArithmetic renders one nonempty input line per question in row-major order.
// It preserves supplied text and never generates questions or answers.
func RenderArithmetic(src []byte, opts ArithmeticOptions) ([]byte, error) {
	layout, err := arithmeticLayoutFor(opts.Layout)
	if err != nil {
		return nil, err
	}
	var questions []string
	for _, line := range strings.Split(strings.TrimPrefix(string(src), "\ufeff"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			questions = append(questions, line)
		}
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("no arithmetic questions provided")
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "宝宝口算"
	}
	if strings.ContainsAny(title, "\r\n") {
		return nil, fmt.Errorf("arithmetic title must be a single line")
	}
	r, err := newRenderer([]byte(title + strings.Join(questions, "")))
	if err != nil {
		return nil, err
	}
	p := r.pdf
	p.SetCellMargin(0)
	width := 190.0 / float64(layout.cols)
	titleSize := 20.0
	if opts.Layout != "" && opts.Layout != "standard" && opts.Layout != "vertical" {
		titleSize = 13
	}
	r.setFont(titleSize)
	if p.GetStringWidth(title) > 190 {
		return nil, fmt.Errorf("arithmetic title is too wide")
	}
	r.setFont(layout.size)
	// Wrap long questions inside their cell; reject overflow rather than silently clipping.
	wrapped := make([][]string, len(questions))
	lineHeight := layout.size * 0.352778 * 1.25
	for i, q := range questions {
		if opts.Layout == "grade4" || opts.Layout == "large-number" || opts.Layout == "negative" {
			q = strings.ReplaceAll(q, "（ ）", "（　　）")
		} else if opts.Layout == "quantity" {
			q = strings.ReplaceAll(q, "（ ）", "（　　　　）")
		}
		wrapped[i] = p.SplitText(q, width-3)
		maxHeight := layout.height - 2
		if opts.Layout == "vertical" {
			maxHeight = 12
		} // Keep the rest for handwritten working.
		if float64(len(wrapped[i]))*lineHeight > maxHeight {
			return nil, fmt.Errorf("question %d is too long for %s layout; shorten it or use a wider layout", i+1, opts.Layout)
		}
	}
	capacity := layout.rows * layout.cols
	for i, lines := range wrapped {
		pos := i % capacity
		if pos == 0 {
			if i > 0 {
				p.AddPage()
			}
			r.setFont(titleSize)
			p.SetXY(10, 7)
			p.CellFormat(190, 10, title, "", 0, "LM", false, 0, "")
			p.SetLineWidth(0.5)
			p.Line(5, 30, 205, 30)
			r.setFont(layout.size)
		}
		x := 10 + float64(pos%layout.cols)*width
		y := 32 + float64(pos/layout.cols)*layout.height
		if opts.Layout != "vertical" {
			y += (layout.height - float64(len(lines))*lineHeight) / 2
		}
		p.SetXY(x, y)
		p.MultiCell(width-3, lineHeight, strings.Join(lines, "\n"), "", "L", false)
	}
	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("output arithmetic pdf: %w", err)
	}
	return buf.Bytes(), nil
}
