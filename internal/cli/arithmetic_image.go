package cli

import (
	"bytes"
	"fmt"
	xdraw "golang.org/x/image/draw"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
)

// rotateArithmeticImage rotates clockwise in memory without changing the file.
func rotateArithmeticImage(src []byte, degrees int) ([]byte, error) {
	if degrees == 0 {
		return src, nil
	}
	if degrees != 90 && degrees != 180 && degrees != 270 {
		return nil, fmt.Errorf("--rotate must be 0, 90, 180, or 270")
	}
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("decode image for rotation failed")
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dw, dh := w, h
	if degrees == 90 || degrees == 270 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x, y
			switch degrees {
			case 90:
				dx, dy = h-1-y, x
			case 180:
				dx, dy = w-1-x, h-1-y
			case 270:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("encode rotated image failed")
	}
	return out.Bytes(), nil
}

// Bound the OCR input resolution without modifying the original photograph.
// This matches the tested document OCR input, avoiding oversized spotting jobs.
func resizeArithmeticImage(src []byte, maxSide int) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("decode OCR input dimensions failed")
	}
	longest := max(cfg.Width, cfg.Height)
	if longest <= maxSide {
		return src, nil
	}
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("decode OCR input failed")
	}
	w, h := max(1, cfg.Width*maxSide/longest), max(1, cfg.Height*maxSide/longest)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("encode OCR input failed")
	}
	return out.Bytes(), nil
}
