package cli

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var htmlHrefRE = regexp.MustCompile(`(?i)\bhref\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)

type cnoteHTMLFormatter struct {
	format    string
	out       []rune
	listStack []string
	linkStack []cnoteLinkState
}

type cnoteLinkState struct {
	href  string
	start int
}

func formatCnoteHTML(body []byte, format string) string {
	f := &cnoteHTMLFormatter{format: format}
	input := string(body)
	for len(input) > 0 {
		tagStart := strings.IndexByte(input, '<')
		if tagStart < 0 {
			f.writeText(input)
			break
		}
		f.writeText(input[:tagStart])
		input = input[tagStart:]

		tagEnd := strings.IndexByte(input, '>')
		if tagEnd < 0 {
			f.writeText(input)
			break
		}
		f.writeTag(input[1:tagEnd])
		input = input[tagEnd+1:]
	}
	return normalizeCnoteText(string(f.out))
}

func (f *cnoteHTMLFormatter) writeTag(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "!") || strings.HasPrefix(raw, "?") {
		return
	}

	closing := strings.HasPrefix(raw, "/")
	if closing {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	}
	raw = strings.TrimSuffix(raw, "/")

	name := strings.ToLower(raw)
	if i := strings.IndexFunc(name, unicode.IsSpace); i >= 0 {
		name = name[:i]
	}

	if closing {
		f.writeClosingTag(name)
		return
	}
	f.writeOpeningTag(name, raw)
}

func (f *cnoteHTMLFormatter) writeOpeningTag(name, raw string) {
	switch name {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		f.blockBreak()
		if f.format == "markdown" {
			level := int(name[1] - '0')
			f.write(strings.Repeat("#", level) + " ")
		}
	case "p", "div", "section", "article":
		f.blockBreak()
	case "br":
		f.lineBreak()
	case "ul", "ol":
		f.blockBreak()
		f.listStack = append(f.listStack, name)
	case "li":
		f.lineBreak()
		if len(f.listStack) > 0 && f.listStack[len(f.listStack)-1] == "ol" {
			f.write("1. ")
		} else {
			f.write("- ")
		}
	case "strong", "b":
		if f.format == "markdown" {
			f.write("**")
		}
	case "em", "i":
		if f.format == "markdown" {
			f.write("*")
		}
	case "a":
		href := html.UnescapeString(htmlTagAttr(raw, "href"))
		if f.format == "markdown" {
			f.write("[")
		}
		f.linkStack = append(f.linkStack, cnoteLinkState{href: href, start: len(f.out)})
	}
}

func (f *cnoteHTMLFormatter) writeClosingTag(name string) {
	switch name {
	case "h1", "h2", "h3", "h4", "h5", "h6", "p", "div", "section", "article":
		f.blockBreak()
	case "ul", "ol":
		if len(f.listStack) > 0 {
			f.listStack = f.listStack[:len(f.listStack)-1]
		}
		f.blockBreak()
	case "li":
		f.lineBreak()
	case "strong", "b":
		if f.format == "markdown" {
			f.write("**")
		}
	case "em", "i":
		if f.format == "markdown" {
			f.write("*")
		}
	case "a":
		if len(f.linkStack) == 0 {
			return
		}
		link := f.linkStack[len(f.linkStack)-1]
		f.linkStack = f.linkStack[:len(f.linkStack)-1]
		if link.href == "" {
			if f.format == "markdown" {
				f.write("]")
			}
			return
		}
		if f.format == "markdown" {
			f.write("](" + link.href + ")")
			return
		}
		visible := strings.TrimSpace(string(f.out[link.start:]))
		if visible != link.href {
			f.write(" (" + link.href + ")")
		}
	}
}

func (f *cnoteHTMLFormatter) writeText(raw string) {
	text := collapseHTMLWhitespace(html.UnescapeString(raw))
	if text == "" {
		return
	}
	if len(f.out) == 0 || isCnoteSpace(f.out[len(f.out)-1]) {
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
	}
	f.write(text)
}

func (f *cnoteHTMLFormatter) write(s string) {
	f.out = append(f.out, []rune(s)...)
}

func (f *cnoteHTMLFormatter) lineBreak() {
	if len(f.out) == 0 || f.out[len(f.out)-1] == '\n' {
		return
	}
	f.write("\n")
}

func (f *cnoteHTMLFormatter) blockBreak() {
	if len(f.out) == 0 {
		return
	}
	newlines := 0
	for i := len(f.out) - 1; i >= 0 && f.out[i] == '\n'; i-- {
		newlines++
	}
	if newlines >= 2 {
		return
	}
	f.write(strings.Repeat("\n", 2-newlines))
}

func htmlTagAttr(raw, key string) string {
	matches := htmlHrefRE.FindStringSubmatch(raw)
	if len(matches) == 0 || !strings.EqualFold(key, "href") {
		return ""
	}
	for _, match := range matches[2:] {
		if match != "" {
			return match
		}
	}
	return ""
}

func collapseHTMLWhitespace(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	hasLeading := len(runes) > 0 && unicode.IsSpace(runes[0])
	hasTrailing := len(runes) > 0 && unicode.IsSpace(runes[len(runes)-1])
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	collapsed := strings.Join(fields, " ")
	if hasLeading {
		collapsed = " " + collapsed
	}
	if hasTrailing {
		collapsed += " "
	}
	return collapsed
}

func normalizeCnoteText(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := 0
	for _, line := range lines {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if strings.TrimSpace(line) == "" {
			blank++
			if blank <= 1 && len(out) > 0 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		out = append(out, line)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func isCnoteSpace(r rune) bool {
	return unicode.IsSpace(r)
}
