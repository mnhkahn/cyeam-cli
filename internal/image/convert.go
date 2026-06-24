package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"slices"

	"github.com/HugoSmits86/nativewebp"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register webp decoder for image.Decode
)

// image.Decode relies on registered decoders; gif/jpeg/png are imported above
// for their encoders and register themselves via their init functions.

// Formats accepted as conversion targets.
var Formats = []string{"jpg", "png", "webp", "gif", "ico", "base64"}

// ValidFormat reports whether format is a supported conversion target.
func ValidFormat(format string) bool {
	return slices.Contains(Formats, format)
}

// Decode reads an image. Bitmap formats (jpg/png/gif/webp) are detected by
// content; SVG has no magic number so isSVG must be set by the caller, in which
// case the vector is rasterized. svgW/svgH set the rasterization canvas when the
// SVG omits an intrinsic size (falls back to the viewBox otherwise).
func Decode(r io.Reader, isSVG bool, svgW, svgH int) (image.Image, error) {
	if isSVG {
		return rasterizeSVG(r, svgW, svgH)
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

func rasterizeSVG(r io.Reader, w, h int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}
	// Ensure a valid viewBox: SetTarget divides by ViewBox.W/H, so an SVG that
	// omits both viewBox and size must fall back to the spec default to avoid a
	// divide-by-zero in the transform.
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		icon.ViewBox.W, icon.ViewBox.H = 300, 150
	}
	vbW, vbH := icon.ViewBox.W, icon.ViewBox.H

	// Default canvas to the SVG's own size; explicit width/height override.
	outW, outH := int(vbW), int(vbH)
	if w > 0 || h > 0 {
		outW, outH = w, h
		if w <= 0 {
			outW = int(vbW * float64(h) / vbH)
		}
		if h <= 0 {
			outH = int(vbH * float64(w) / vbW)
		}
	}
	outW, outH = max(outW, 1), max(outH, 1)

	// SetTarget maps the viewBox onto the full output rect; the scanner must be
	// sized to that output rect (not the viewBox) or rendering clips when the
	// canvas is larger than the source viewBox.
	icon.SetTarget(0, 0, float64(outW), float64(outH))
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	scanner := rasterx.NewScannerGV(outW, outH, dst, dst.Bounds())
	raster := rasterx.NewDasher(outW, outH, scanner)
	icon.Draw(raster, 1.0)
	return dst, nil
}

// Resize scales img to the requested width/height. A zero dimension means
// "derive from the other"; when keepRatio is set the missing dimension follows
// the source aspect ratio. If both are zero the image is returned unchanged.
// Mirrors the sizing logic in the web converter (picconverter.html convertImage).
func Resize(img image.Image, w, h int, keepRatio bool) image.Image {
	b := img.Bounds()
	ow, oh := b.Dx(), b.Dy()
	if ow == 0 || oh == 0 {
		return img
	}
	ratio := float64(oh) / float64(ow)

	newW, newH := ow, oh
	switch {
	case w > 0 && h > 0:
		newW, newH = w, h
	case w > 0:
		newW = w
		if keepRatio {
			newH = int(float64(w) * ratio)
		}
	case h > 0:
		newH = h
		if keepRatio {
			newW = int(float64(h) / ratio)
		}
	default:
		return img
	}
	// Truncating the derived dimension for extreme aspect ratios can reach zero
	// (e.g. 1000x1 with --width 1); encoders reject zero-sized images.
	newW, newH = max(newW, 1), max(newH, 1)
	if newW == ow && newH == oh {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// Encode serializes img to the target format. quality (1-100) applies to jpg
// only; webp here is lossless (VP8L) so quality is ignored. The base64 result
// is a PNG data URL string. For ico use EncodeICO.
func Encode(img image.Image, format string, quality int) ([]byte, error) {
	switch format {
	case "jpg":
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpg: %w", err)
		}
		return buf.Bytes(), nil
	case "png":
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode png: %w", err)
		}
		return buf.Bytes(), nil
	case "gif":
		var buf bytes.Buffer
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, fmt.Errorf("encode gif: %w", err)
		}
		return buf.Bytes(), nil
	case "webp":
		var buf bytes.Buffer
		if err := nativewebp.Encode(&buf, img, nil); err != nil {
			return nil, fmt.Errorf("encode webp: %w", err)
		}
		return buf.Bytes(), nil
	case "ico":
		return EncodeICO(img)
	case "base64":
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode png for base64: %w", err)
		}
		dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		return []byte(dataURL), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
