package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"

	xdraw "golang.org/x/image/draw"
)

// icoSizes are the standard favicon dimensions embedded in the .ico container,
// matching the web converter (picconverter.html createICOFromPNGs).
var icoSizes = []int{16, 32, 48}

// EncodeICO builds a multi-size .ico file. It rescales src into 16/32/48 px
// PNG frames and packs them into the ICONDIR/ICONDIRENTRY structure.
func EncodeICO(src image.Image) ([]byte, error) {
	type frame struct {
		size int
		png  []byte
	}
	frames := make([]frame, 0, len(icoSizes))
	for _, size := range icoSizes {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
		var buf bytes.Buffer
		if err := png.Encode(&buf, dst); err != nil {
			return nil, fmt.Errorf("encode ico frame %dpx: %w", size, err)
		}
		frames = append(frames, frame{size: size, png: buf.Bytes()})
	}

	var out bytes.Buffer
	// ICONDIR header (6 bytes): reserved=0, type=1 (icon), count.
	binary.Write(&out, binary.LittleEndian, uint16(0))
	binary.Write(&out, binary.LittleEndian, uint16(1))
	binary.Write(&out, binary.LittleEndian, uint16(len(frames)))

	// Image data starts after the header and all directory entries.
	offset := 6 + 16*len(frames)
	for _, f := range frames {
		dim := byte(f.size)
		if f.size >= 256 {
			dim = 0 // 0 means 256 in the ICO format
		}
		out.WriteByte(dim)                                          // width
		out.WriteByte(dim)                                          // height
		out.WriteByte(0)                                            // color palette
		out.WriteByte(0)                                            // reserved
		binary.Write(&out, binary.LittleEndian, uint16(1))         // color planes
		binary.Write(&out, binary.LittleEndian, uint16(32))        // bits per pixel
		binary.Write(&out, binary.LittleEndian, uint32(len(f.png))) // data size
		binary.Write(&out, binary.LittleEndian, uint32(offset))     // data offset
		offset += len(f.png)
	}
	for _, f := range frames {
		out.Write(f.png)
	}
	return out.Bytes(), nil
}
