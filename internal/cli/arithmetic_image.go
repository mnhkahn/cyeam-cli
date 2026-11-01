package cli

import (
	"bytes"
	"fmt"
	"image"
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
