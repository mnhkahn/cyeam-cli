package cli

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestArithmeticRotationPreservesPixelsAndSource(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	src.Set(0, 0, color.NRGBA{R: 255, A: 255})
	src.Set(1, 2, color.NRGBA{B: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, src); err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), encoded.Bytes()...)
	for _, tc := range []struct{ angle, w, h, rx, ry, bx, by int }{{90, 3, 2, 2, 0, 0, 1}, {180, 2, 3, 1, 2, 0, 0}, {270, 3, 2, 0, 1, 2, 0}} {
		data, err := rotateArithmeticImage(original, tc.angle)
		if err != nil {
			t.Fatal(err)
		}
		rotated, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		red := color.NRGBAModel.Convert(rotated.At(tc.rx, tc.ry)).(color.NRGBA)
		blue := color.NRGBAModel.Convert(rotated.At(tc.bx, tc.by)).(color.NRGBA)
		if rotated.Bounds().Dx() != tc.w || rotated.Bounds().Dy() != tc.h || red.R != 255 || blue.B != 255 {
			t.Fatal("rotation changed pixel positions")
		}
	}
	if !bytes.Equal(original, encoded.Bytes()) {
		t.Fatal("original changed")
	}
}
