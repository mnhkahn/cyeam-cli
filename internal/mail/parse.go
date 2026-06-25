package mail

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // register non-UTF-8 charsets
	gomail "github.com/emersion/go-message/mail"
)

// Message is a parsed email with decoded headers and plain-text body.
type Message struct {
	From    string    `json:"from"`
	To      string    `json:"to"`
	Cc      string    `json:"cc,omitempty"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Body    string    `json:"body"`
}

// ParseMessage decodes an RFC822 message: headers plus a plain-text body
// (preferring text/plain, falling back to a stripped text/html).
func ParseMessage(r io.Reader) (*Message, error) {
	mr, err := gomail.CreateReader(r)
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, fmt.Errorf("read message: %w", err)
	}
	defer mr.Close()

	m := &Message{}
	m.From = addrList(mr.Header.AddressList("From"))
	m.To = addrList(mr.Header.AddressList("To"))
	m.Cc = addrList(mr.Header.AddressList("Cc"))
	if subj, err := mr.Header.Subject(); err == nil {
		m.Subject = subj
	}
	if date, err := mr.Header.Date(); err == nil {
		m.Date = date
	}

	var plain, html string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if message.IsUnknownCharset(err) {
				continue
			}
			break
		}
		switch p.Header.(type) {
		case *gomail.InlineHeader:
			ct := p.Header.Get("Content-Type")
			b, _ := io.ReadAll(p.Body)
			if (ct == "" || strings.HasPrefix(ct, "text/plain")) && plain == "" {
				plain = string(b)
			} else if strings.HasPrefix(ct, "text/html") && html == "" {
				html = string(b)
			}
		}
	}

	if plain != "" {
		m.Body = strings.TrimSpace(plain)
	} else {
		m.Body = strings.TrimSpace(stripHTML(html))
	}
	return m, nil
}

func addrList(addrs []*gomail.Address, err error) string {
	if err != nil {
		return ""
	}
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", a.Name, a.Address))
		} else {
			parts = append(parts, a.Address)
		}
	}
	return strings.Join(parts, ", ")
}

// stripHTML removes tags for a rough plain-text fallback when no text/plain
// part exists. Not a full renderer — just enough to make HTML-only mail
// readable.
func stripHTML(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	// Collapse runs of blank lines.
	lines := strings.Split(b.String(), "\n")
	var out []string
	for _, ln := range lines {
		if t := strings.TrimSpace(ln); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, "\n")
}
