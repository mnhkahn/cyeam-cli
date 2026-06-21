package pinyin

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/mnhkahn/gofpdf"
	"github.com/mozillazg/go-pinyin"
)

//go:embed font/pinyin-wenkai-light.ttf
var pyFont []byte

func GenerateSheetPDF(text string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("pyfont", "", pyFont)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	pdf.SetFont("pyfont", "", 20)
	pdf.SetXY(10, 7)
	pdf.CellFormat(190, 10, "看拼音写字", "", 0, "C", false, 0, "")

	const paddingLeft = 13
	xStart := float64(paddingLeft)
	yStart := float64(20)
	const wMi = float64(11)
	const hPy = float64(7)

	argsPY := pinyin.NewArgs()
	argsPY.Style = pinyin.Tone

	for _, word := range strings.Split(text, " ") {
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
		if yStart >= 236 {
			break
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

		charIdx := 0
		for _, r := range chars {
			if !unicode.Is(unicode.Han, r) {
				continue
			}
			x := xStart + float64(charIdx)*wMi

			// pinyin above cell
			py := pinyin.Pinyin(string(r), argsPY)
			if len(py) > 0 && len(py[0]) > 0 {
				pdf.SetFont("pyfont", "", 8)
				pdf.SetXY(x, yStart)
				pdf.CellFormat(wMi, hPy, py[0][0], "", 0, "C", false, 0, "")
			}

			// vertical dashed midline per cell
			pdf.SetLineWidth(0.15)
			pdf.Line(x+wMi/2, cy, x+wMi/2, cy+wMi)

			// diagonals per cell
			pdf.Line(x, cy, x+wMi, cy+wMi)
			pdf.Line(x+wMi, cy, x, cy+wMi)

			charIdx++
		}

		pdf.SetDashPattern([]float64{}, 0)

		xStart += blockWidth + 5
	}

	pdf.SetFont("pyfont", "", 13)
	pdf.SetXY(10, 237)
	pdf.CellFormat(70, 10, "改错：", "", 0, "L", false, 0, "")

	for i := 0; i < 4; i++ {
		y := 255 + float64(i)*10
		pdf.SetLineWidth(0.1)
		pdf.Line(10, y, 200, y)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return buf.Bytes(), nil
}
