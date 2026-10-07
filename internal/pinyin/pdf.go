package pinyin

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/mnhkahn/go-pinyin"
	"github.com/mnhkahn/gofpdf"
)

//go:embed font/pinyin-wenkai-light.ttf
var pyFont []byte

func GenerateSheetPDF(text string) ([]byte, error) {
	return GenerateSheetPDFWithHeaderFont(text, pyFont)
}

// GenerateSheetPDFWithHeaderFont uses a Chinese-capable font for worksheet labels.
func GenerateSheetPDFWithHeaderFont(text string, headerFont []byte) ([]byte, error) {
	words := sheetWords(text)
	if len(words) == 0 {
		return nil, fmt.Errorf("no Chinese characters provided")
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("pyfont", "", pyFont)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddUTF8FontFromBytes("header", "", headerFont)
	addPage := func() {
		pdf.AddPage()
		pdf.SetFont("header", "", 20)
		pdf.SetXY(10, 7)
		pdf.CellFormat(190, 10, "看拼音写字", "", 0, "L", false, 0, "")
		pdf.SetFont("header", "", 13)
		pdf.SetXY(10, 237)
		pdf.CellFormat(70, 10, "改错：", "", 0, "L", false, 0, "")
		pdf.SetLineWidth(0.1)
		for i := 0; i < 4; i++ {
			pdf.Line(10, 255+float64(i)*10, 200, 255+float64(i)*10)
		}
	}
	addPage()

	const paddingLeft = 13
	xStart := float64(paddingLeft)
	yStart := float64(20)
	const wMi = float64(11)
	const hPy = float64(7)

	for _, word := range words {
		if word == "" {
			continue
		}

		chars := []rune(word)
		chineseCount := 0
		for _, r := range chars {
			if unicode.Is(unicode.Han, r) {
				chineseCount++
			}
		}
		if chineseCount == 0 {
			continue
		}
		blockWidth := float64(11 * chineseCount)

		if xStart+blockWidth > 200 {
			xStart = paddingLeft
			yStart += wMi + hPy
		}
		if yStart+wMi+hPy > 236 {
			addPage()
			xStart = paddingLeft
			yStart = 20
		}

		cy := yStart + hPy

		// outer rectangle for entire block
		pdf.SetLineWidth(0.3)
		pdf.Rect(xStart, cy, blockWidth, wMi, "D")
		pdf.SetDashPattern([]float64{0.8, 0.8}, 0)

		// vertical dividers between characters
		pdf.SetLineWidth(0.15)
		for i := 1; i < chineseCount; i++ {
			x := xStart + float64(i)*wMi
			pdf.Line(x, cy, x, cy+wMi)
		}

		// horizontal dashed midline across the block
		pdf.Line(xStart, cy+wMi/2, xStart+blockWidth, cy+wMi/2)

		// 使用独立的 go-pinyin 包转换拼音
		pyList := pinyin.WithTone(word)

		charIdx := 0
		pyIdx := 0
		for _, r := range chars {
			if !unicode.Is(unicode.Han, r) {
				continue
			}
			x := xStart + float64(charIdx)*wMi

			// pinyin above cell
			if pyIdx < len(pyList) {
				pdf.SetFont("pyfont", "", 8)
				pdf.SetXY(x, yStart)
				pdf.CellFormat(wMi, hPy, pyList[pyIdx], "", 0, "C", false, 0, "")
			}

			// vertical dashed midline per cell
			pdf.SetLineWidth(0.15)
			pdf.Line(x+wMi/2, cy, x+wMi/2, cy+wMi)

			// diagonals per cell
			pdf.Line(x, cy, x+wMi, cy+wMi)
			pdf.Line(x+wMi, cy, x, cy+wMi)

			charIdx++
			pyIdx++
		}

		pdf.SetDashPattern([]float64{}, 0)

		xStart += blockWidth + 5
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

// Split on non-Han separators and cap each block at the page width (17 cells).
func sheetWords(text string) []string {
	var words []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.Is(unicode.Han, r) }) {
		chars := []rune(word)
		for len(chars) > 17 {
			words = append(words, string(chars[:17]))
			chars = chars[17:]
		}
		if len(chars) > 0 {
			words = append(words, string(chars))
		}
	}
	return words
}
