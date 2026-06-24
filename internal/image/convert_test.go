package image

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	_ "golang.org/x/image/webp"
)

func testImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	return img
}

func TestEncodeDecodableFormats(t *testing.T) {
	src := testImage(64, 48)
	cases := []struct {
		format string
		decode func([]byte) (image.Image, error)
	}{
		{"jpg", func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }},
		{"png", func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }},
		{"gif", func(b []byte) (image.Image, error) { return gif.Decode(bytes.NewReader(b)) }},
		{"webp", func(b []byte) (image.Image, error) { i, _, e := image.Decode(bytes.NewReader(b)); return i, e }},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			data, err := Encode(src, tc.format, 90)
			if err != nil {
				t.Fatalf("encode %s: %v", tc.format, err)
			}
			if len(data) == 0 {
				t.Fatalf("encode %s: empty output", tc.format)
			}
			got, err := tc.decode(data)
			if err != nil {
				t.Fatalf("decode %s back: %v", tc.format, err)
			}
			if got.Bounds().Dx() != 64 || got.Bounds().Dy() != 48 {
				t.Errorf("%s roundtrip size = %v, want 64x48", tc.format, got.Bounds())
			}
		})
	}
}

func TestEncodeBase64(t *testing.T) {
	data, err := Encode(testImage(8, 8), "base64", 90)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "data:image/png;base64,") {
		t.Errorf("base64 output missing data URL prefix: %q", string(data)[:32])
	}
}

func TestResize(t *testing.T) {
	src := testImage(100, 50) // ratio h/w = 0.5
	t.Run("width only keeps ratio", func(t *testing.T) {
		out := Resize(src, 40, 0, true)
		if out.Bounds().Dx() != 40 || out.Bounds().Dy() != 20 {
			t.Errorf("got %v, want 40x20", out.Bounds())
		}
	})
	t.Run("height only keeps ratio", func(t *testing.T) {
		out := Resize(src, 0, 10, true)
		if out.Bounds().Dx() != 20 || out.Bounds().Dy() != 10 {
			t.Errorf("got %v, want 20x10", out.Bounds())
		}
	})
	t.Run("both dimensions exact", func(t *testing.T) {
		out := Resize(src, 30, 30, true)
		if out.Bounds().Dx() != 30 || out.Bounds().Dy() != 30 {
			t.Errorf("got %v, want 30x30", out.Bounds())
		}
	})
	t.Run("zero returns original", func(t *testing.T) {
		out := Resize(src, 0, 0, true)
		if out.Bounds() != src.Bounds() {
			t.Errorf("got %v, want unchanged %v", out.Bounds(), src.Bounds())
		}
	})
}

func TestEncodeICOHeader(t *testing.T) {
	data, err := EncodeICO(testImage(64, 64))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 6 {
		t.Fatal("ico too short")
	}
	// ICONDIR: reserved=0, type=1, count=3 (little-endian uint16 each).
	if data[0] != 0 || data[1] != 0 {
		t.Errorf("reserved bytes = %v, want 0 0", data[:2])
	}
	if data[2] != 1 || data[3] != 0 {
		t.Errorf("type = %v, want 1 0 (icon)", data[2:4])
	}
	if data[4] != byte(len(icoSizes)) || data[5] != 0 {
		t.Errorf("image count = %v, want %d 0", data[4:6], len(icoSizes))
	}
}

func TestRasterizeSVG(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="red"/></svg>`
	out, err := Decode(strings.NewReader(svg), true, 200, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 200 || out.Bounds().Dy() != 200 {
		t.Errorf("svg raster size = %v, want 200x200", out.Bounds())
	}
}

func TestRasterizeSVGUpscaleFills(t *testing.T) {
	// viewBox 100x100, upscaled to 200x200: rendering must fill the whole canvas
	// (a clipping bug would leave the bottom-right 3 quadrants transparent).
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="red"/></svg>`
	out, err := Decode(strings.NewReader(svg), true, 200, 200)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, a := out.At(180, 180).RGBA()
	if a == 0 {
		t.Error("bottom-right pixel is transparent: SVG rendering clipped to original viewBox")
	}
}

func TestRasterizeSVGNoViewBox(t *testing.T) {
	// No viewBox and no explicit size must fall back to 300x150 without a
	// divide-by-zero in SetTarget.
	const svg = `<svg xmlns="http://www.w3.org/2000/svg"><rect width="50" height="50" fill="green"/></svg>`
	out, err := Decode(strings.NewReader(svg), true, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 300 || out.Bounds().Dy() != 150 {
		t.Errorf("no-viewBox default size = %v, want 300x150", out.Bounds())
	}
}

func TestResizeExtremeRatioClamps(t *testing.T) {
	// 1000x1 down to width 1 would derive height 0; must clamp to 1.
	out := Resize(testImage(1000, 1), 1, 0, true)
	if out.Bounds().Dy() < 1 {
		t.Errorf("derived height = %d, want >= 1", out.Bounds().Dy())
	}
	// And the output must still be encodable.
	if _, err := Encode(out, "png", 90); err != nil {
		t.Errorf("encode clamped image: %v", err)
	}
}

func TestValidFormat(t *testing.T) {
	for _, f := range Formats {
		if !ValidFormat(f) {
			t.Errorf("ValidFormat(%q) = false", f)
		}
	}
	if ValidFormat("tiff") {
		t.Error("ValidFormat(tiff) = true, want false")
	}
}
