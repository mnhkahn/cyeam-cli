package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flopp/go-findfont"
	"github.com/mnhkahn/gofpdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//go:embed font/pinyin-wenkai-light.ttf
var pdfFont []byte

func findChineseFont() []byte {
	// Try common Chinese fonts in order of preference
	fonts := []string{
		"Arial Unicode",
		"Hiragino Sans GB",
		"STHeiti",
		"Heiti SC",
		"PingFang",
		"Songti",
		"Microsoft YaHei",
		"SimHei",
		"SimSun",
		"Noto Sans CJK SC",
		"Noto Sans CJK TC",
	}
	for _, font := range fonts {
		if fontBytes, err := loadFont(font); err == nil {
			return fontBytes
		}
	}
	// Fallback: list all fonts and find any with CJK in name
	for _, path := range findfont.List() {
		if strings.Contains(strings.ToLower(path), "cjk") || strings.Contains(strings.ToLower(path), "chinese") {
			if fontBytes, err := loadFontByPath(path); err == nil {
				return fontBytes
			}
		}
	}
	// Last resort: use embedded fallback font
	return pdfFont
}

func loadFont(fontName string) ([]byte, error) {
	path, err := findfont.Find(fontName)
	if err != nil {
		return nil, err
	}
	return loadFontByPath(path)
}

func loadFontByPath(path string) ([]byte, error) {
	// Ensure absolute path
	if !filepath.IsAbs(path) {
		path = "/" + path
	}
	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = realPath
	}
	// Read font file bytes
	return os.ReadFile(path)
}

const (
	marginLeft   = 20.0
	marginRight  = 20.0
	marginTop    = 20.0
	marginBottom = 20.0
	contentW     = 210.0 - marginLeft - marginRight
	contentH     = 297.0 - marginTop - marginBottom
)

type renderer struct {
	pdf *gofpdf.Fpdf
	y   float64
}

func newRenderer() *renderer {
	fontBytes := findChineseFont()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	pdf.AddUTF8FontFromBytes("pdf", "", fontBytes)
	if pdf.Ok() {
		// Try setting the font to verify it works
		pdf.SetFont("pdf", "", 12)
		if !pdf.Ok() {
			// System font failed, fall back to embedded font
			pdf = gofpdf.New("P", "mm", "A4", "")
			pdf.SetAutoPageBreak(false, 0)
			pdf.AddPage()
			pdf.AddUTF8FontFromBytes("pdf", "", pdfFont)
			pdf.SetFont("pdf", "", 12)
		}
	}
	return &renderer{pdf: pdf, y: marginTop}
}

func (r *renderer) ensurePage(h float64) {
	if r.y+h > marginTop+contentH {
		r.pdf.AddPage()
		r.y = marginTop
	}
}

func (r *renderer) setFont(size float64) {
	r.pdf.SetFont("pdf", "", size)
}

func RenderMarkdown(src []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.TaskList),
	)
	doc := md.Parser().Parse(text.NewReader(src))
	r := newRenderer()
	walkMarkdown(r, doc, src)
	var buf bytes.Buffer
	if err := r.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func walkMarkdown(r *renderer, node ast.Node, src []byte) {
	switch n := node.(type) {
	case *ast.Document:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkMarkdown(r, child, src)
		}
	case *ast.Heading:
		var sizes = map[int]float64{1: 22, 2: 16, 3: 13, 4: 11}
		size := sizes[n.Level]
		if size == 0 {
			size = 11
		}
		r.ensurePage(size + 6)
		r.setFont(size)
		r.pdf.SetXY(marginLeft, r.y)
		var line strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			line.WriteString(collectText(child, src))
		}
		r.pdf.MultiCell(contentW, size*0.35, line.String(), "", "L", false)
		r.y = r.pdf.GetY() + 3
	case *ast.Paragraph:
		r.ensurePage(14)
		r.setFont(11)
		r.pdf.SetXY(marginLeft, r.y)
		var line strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			line.WriteString(collectText(child, src))
		}
		r.pdf.MultiCell(contentW, 5, line.String(), "", "L", false)
		r.y = r.pdf.GetY() + 2
	case *ast.FencedCodeBlock:
		r.renderCodeBlock(n, src)
	case *ast.CodeBlock:
		r.renderCodeBlock(n, src)
	case *ast.List:
		idx := n.Start
		if idx < 1 {
			idx = 1
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkList(r, child, src, n.IsOrdered(), n.Start, &idx)
		}
	case *ast.ListItem:
		// List items are handled by walkList
	case *ast.ThematicBreak:
		r.ensurePage(10)
		r.y += 3
		r.pdf.SetY(r.y)
		r.pdf.Line(marginLeft, r.y, marginLeft+contentW, r.y)
		r.y += 5
	case *ast.Text, *ast.String, *ast.CodeSpan, *ast.Emphasis, *ast.Link:
		// Leaf nodes or nodes whose text content is collected by parents
	default:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkMarkdown(r, child, src)
		}
	}
}

func walkList(r *renderer, node ast.Node, src []byte, ordered bool, start int, idx *int) {
	switch n := node.(type) {
	case *ast.ListItem:
		r.ensurePage(11)
		r.setFont(11)
		prefix := "・ "
		if ordered {
			prefix = fmt.Sprintf("%d. ", *idx)
			*idx++
		}
		r.pdf.SetXY(marginLeft+5, r.y)
		r.pdf.Cell(8, 5, prefix)
		r.pdf.SetXY(marginLeft+13, r.y)

		var line strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if _, ok := child.(*ast.List); ok {
				continue
			}
			line.WriteString(collectText(child, src))
		}
		r.pdf.MultiCell(contentW-13, 5, line.String(), "", "L", false)
		r.y = r.pdf.GetY() + 1

		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkList(r, child, src, ordered, start, idx)
		}

	case *ast.List:
		subIdx := 1
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkList(r, child, src, n.IsOrdered(), n.Start, &subIdx)
		}
	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			walkList(r, child, src, ordered, start, idx)
		}
	}
}

func (r *renderer) renderCodeBlock(n ast.Node, src []byte) {
	var line strings.Builder
	switch node := n.(type) {
	case *ast.FencedCodeBlock:
		for i := 0; i < node.Lines().Len(); i++ {
			seg := node.Lines().At(i)
			line.Write(seg.Value(src))
		}
	case *ast.CodeBlock:
		for i := 0; i < node.Lines().Len(); i++ {
			seg := node.Lines().At(i)
			line.Write(seg.Value(src))
		}
	}
	text := strings.TrimRight(line.String(), "\n")
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	lineH := 4.5
	h := float64(len(lines))*lineH + 6
	r.ensurePage(h)

	r.pdf.SetY(r.y)
	r.pdf.SetFillColor(245, 245, 245)
	r.pdf.Rect(marginLeft, r.y, contentW, h, "F")
	r.pdf.SetFillColor(0, 0, 0)

	r.setFont(9)
	cy := r.y + 3
	for _, l := range lines {
		r.pdf.SetXY(marginLeft+4, cy)
		r.pdf.Cell(contentW-8, lineH, strings.TrimRight(l, "\r\n"))
		cy += lineH
	}
	r.y = cy + 4
	r.pdf.SetY(r.y)
}

func collectText(node ast.Node, src []byte) string {
	var buf strings.Builder
	walkText(node, src, &buf)
	return buf.String()
}

func walkText(node ast.Node, src []byte, buf *strings.Builder) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(src))
		if n.SoftLineBreak() {
			buf.WriteString(" ")
		}
	case *east.TaskCheckBox:
		if n.IsChecked {
			buf.WriteString("[x] ")
		} else {
			buf.WriteString("[ ] ")
		}
	case *ast.CodeSpan:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				buf.WriteString("`")
				buf.Write(t.Segment.Value(src))
				buf.WriteString("`")
			}
		}
	case *ast.Emphasis:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkText(child, src, buf)
		}
	case *ast.Link:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkText(child, src, buf)
		}
	case *ast.String:
		buf.WriteString(string(n.Value))
	default:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			walkText(child, src, buf)
		}
	}
}

func RenderHTML(src []byte) ([]byte, error) {
	md := htmlToMarkdown(src)
	return RenderMarkdown(md)
}

func htmlToMarkdown(src []byte) []byte {
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return src
	}
	var buf bytes.Buffer
	walkHTML(doc, &buf)
	return buf.Bytes()
}

func walkHTML(n *html.Node, buf *bytes.Buffer) {
	switch n.Type {
	case html.TextNode:
		buf.WriteString(n.Data)
	case html.ElementNode:
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level := int(n.DataAtom - atom.H1 + 1)
			if level > 4 {
				level = 4
			}
			buf.WriteString("\n\n")
			for i := 0; i < level; i++ {
				buf.WriteString("#")
			}
			buf.WriteString(" ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			buf.WriteString("\n\n")
			return
		case atom.P:
			buf.WriteString("\n\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			buf.WriteString("\n\n")
			return
		case atom.Br:
			buf.WriteString("\n")
			return
		case atom.Strong, atom.B:
			buf.WriteString("**")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			buf.WriteString("**")
			return
		case atom.Em, atom.I:
			buf.WriteString("*")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			buf.WriteString("*")
			return
		case atom.Code:
			buf.WriteString("`")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			buf.WriteString("`")
			return
		case atom.Pre:
			buf.WriteString("\n\n```\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTMLPlainText(c, buf)
			}
			buf.WriteString("\n```\n\n")
			return
		case atom.Ul:
			buf.WriteString("\n\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.DataAtom == atom.Li {
					buf.WriteString("- ")
					for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
						walkHTML(cc, buf)
					}
					buf.WriteString("\n")
				}
			}
			buf.WriteString("\n")
			return
		case atom.Ol:
			buf.WriteString("\n\n")
			i := 1
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.DataAtom == atom.Li {
					fmt.Fprintf(buf, "%d. ", i)
					i++
					for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
						walkHTML(cc, buf)
					}
					buf.WriteString("\n")
				}
			}
			buf.WriteString("\n")
			return
		case atom.A:
			var textBuf bytes.Buffer
			var href string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, &textBuf)
			}
			linkText := strings.TrimSpace(textBuf.String())
			if linkText != "" {
				buf.WriteString("[")
				buf.WriteString(linkText)
				buf.WriteString("](")
				buf.WriteString(href)
				buf.WriteString(")")
			}
			return
		case atom.Hr:
			buf.WriteString("\n\n---\n\n")
			return
		case atom.Div, atom.Span, atom.Section, atom.Article, atom.Header, atom.Footer, atom.Main, atom.Nav, atom.Aside:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			return
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkHTML(c, buf)
			}
			return
		}
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkHTML(c, buf)
		}
		return
	}
}

func walkHTMLPlainText(n *html.Node, buf *bytes.Buffer) {
	switch n.Type {
	case html.TextNode:
		buf.WriteString(n.Data)
	case html.ElementNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkHTMLPlainText(c, buf)
		}
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkHTMLPlainText(c, buf)
		}
	}
}

func IsHTML(src []byte) bool {
	s := string(bytes.TrimLeft(src, " \t\r\n"))
	return strings.HasPrefix(s, "<!DOCTYPE") || strings.HasPrefix(s, "<html") || strings.HasPrefix(s, "<!doctype")
}
