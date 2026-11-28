package wiring

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed interactive.js
var interaction string

// Render annotates a breadboard SVG with the connections saved in the Fritzing sketch.
func Render(ctx context.Context, sketchPath, baseSVGPath string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(sketchPath))
	if ext != ".fzz" && ext != ".fz" {
		return nil, fmt.Errorf("interactive SVG requires a Fritzing .fzz or .fz sketch")
	}
	diagram, err := readSketch(sketchPath)
	if err != nil {
		return nil, err
	}
	var svg []byte
	if baseSVGPath != "" {
		svg, err = os.ReadFile(baseSVGPath)
	} else {
		svg, err = renderBreadboard(ctx, sketchPath)
	}
	if err != nil {
		return nil, err
	}
	return interactiveSVG(svg, diagram)
}

func renderBreadboard(ctx context.Context, sketchPath string) ([]byte, error) {
	binary, err := exec.LookPath("fritzing-render")
	if err != nil {
		return nil, fmt.Errorf("fritzing-render was not found in PATH; install it or pass --base-svg with a matching breadboard SVG")
	}
	path, err := filepath.Abs(sketchPath)
	if err != nil {
		return nil, err
	}
	result, err := exec.CommandContext(ctx, binary, "render", path, "--view", "breadboard").Output()
	if failure, ok := err.(*exec.ExitError); ok {
		return nil, fmt.Errorf("fritzing-render: %w: %s", failure, strings.TrimSpace(string(failure.Stderr)))
	}
	return result, err
}

type svgStroke struct {
	Insert int
	Ends   [2]point
}

var number = regexp.MustCompile(`[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?`)

func svgStrokes(data []byte) ([]svgStroke, int, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var strokes []svgStroke
	var depth, wireDepth, endTag int
	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read breadboard SVG: %w", err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 && token.Name.Local != "svg" {
				return nil, 0, fmt.Errorf("expected a breadboard SVG")
			}
			attrs := make(map[string]string, len(token.Attr))
			for _, attr := range token.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			if attrs["id"] == "cyeam-wire-routes" {
				return nil, 0, fmt.Errorf("the base SVG already has wiring interactions; provide an original breadboard export")
			}
			if token.Name.Local == "g" && attrs["id"] == "wires" {
				if wireDepth != 0 || len(strokes) != 0 {
					return nil, 0, fmt.Errorf("expected a single fritzing-render wire layer")
				}
				wireDepth = depth
			}
			if wireDepth == 0 || depth != wireDepth+1 {
				continue
			}
			var values []string
			switch token.Name.Local {
			case "line":
				values = []string{attrs["x1"], attrs["y1"], attrs["x2"], attrs["y2"]}
			case "path":
				values = number.FindAllString(attrs["d"], -1)
				if len(values) != 8 || !strings.HasPrefix(attrs["d"], "M") || !strings.Contains(attrs["d"], "C") {
					return nil, 0, fmt.Errorf("unexpected wire path in breadboard SVG")
				}
				values = append(values[:2:2], values[6:]...)
			default:
				continue
			}
			offset := int(decoder.InputOffset()) - 1
			if data[offset-1] == '/' {
				offset--
			}
			stroke := svgStroke{Insert: offset}
			for i, v := range values {
				position, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, 0, fmt.Errorf("invalid wire coordinate %q", v)
				}
				stroke.Ends[i/2][i%2] = position
			}
			strokes = append(strokes, stroke)
		case xml.EndElement:
			if depth == wireDepth {
				wireDepth = 0
			}
			if depth == 1 {
				endTag = before
			}
			depth--
		}
	}
	return strokes, endTag, nil
}

func interactiveSVG(svg []byte, diagram sketch) ([]byte, error) {
	strokes, endTag, err := svgStrokes(svg)
	if err != nil {
		return nil, err
	}
	count := len(diagram.Wires)
	// The upstream breadboard renderer draws all shadows, then all wire strokes, in sketch order.
	if endTag == 0 || len(strokes) != 2*count {
		return nil, fmt.Errorf("SVG wire count does not match the sketch (%d strokes for %d wires); export breadboard view again", len(strokes), count)
	}
	strokes = strokes[count:]
	const scale = 1000.0 / 90.0
	origin := point{
		strokes[0].Ends[0][0] - diagram.Wires[0].Ends[0][0]*scale,
		strokes[0].Ends[0][1] - diagram.Wires[0].Ends[0][1]*scale,
	}
	for i, stroke := range strokes {
		for end, p := range stroke.Ends {
			for axis, v := range p {
				expected := diagram.Wires[i].Ends[end][axis]*scale + origin[axis]
				// Qt rounds coordinates to six significant figures in the exported SVG.
				if math.Abs(v-expected) > .12+math.Abs(v)*.000001 {
					return nil, fmt.Errorf("SVG wire %d does not match the sketch; export breadboard view again", i+1)
				}
			}
		}
	}
	routes := diagram.routes()
	for i := range routes {
		for j := range routes[i].Pins {
			for axis, v := range routes[i].Pins[j].At {
				routes[i].Pins[j].At[axis] = v*scale + origin[axis]
			}
		}
	}
	model, err := json.Marshal(routes)
	if err != nil {
		return nil, err
	}
	var result bytes.Buffer
	last := 0
	for i, stroke := range strokes {
		result.Write(svg[last:stroke.Insert])
		fmt.Fprintf(&result, ` data-cyeam-wire="%d"`, i)
		last = stroke.Insert
	}
	result.Write(svg[last:endTag])
	result.WriteString("\n<metadata id=\"cyeam-wire-routes\">")
	if err := xml.EscapeText(&result, model); err != nil {
		return nil, err
	}
	result.WriteString("</metadata>\n<script type=\"application/ecmascript\"><![CDATA[\n")
	result.WriteString(interaction)
	result.WriteString("\n]]></script>\n")
	result.Write(svg[endTag:])
	return result.Bytes(), nil
}
