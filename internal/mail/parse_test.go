package mail

import (
	"strings"
	"testing"
)

func TestParseMessagePlainText(t *testing.T) {
	// Subject uses RFC2047 quoted-printable UTF-8 (Chinese); body is text/plain.
	raw := "From: \"Li Chao\" <lichao@cyeam.com>\r\n" +
		"To: me@example.com\r\n" +
		"Subject: =?UTF-8?B?5rWL6K+V6YKu5Lu2?=\r\n" +
		"Date: Mon, 02 Jan 2006 15:04:05 +0800\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"你好，这是正文。\r\n"

	m, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "测试邮件" {
		t.Errorf("Subject = %q, want 测试邮件", m.Subject)
	}
	if !strings.Contains(m.From, "lichao@cyeam.com") {
		t.Errorf("From = %q, want to contain lichao@cyeam.com", m.From)
	}
	if !strings.Contains(m.Body, "这是正文") {
		t.Errorf("Body = %q, want to contain 这是正文", m.Body)
	}
}

func TestParseMessageMultipartPrefersPlain(t *testing.T) {
	raw := "From: a@b.com\r\n" +
		"To: c@d.com\r\n" +
		"Subject: hi\r\n" +
		"Content-Type: multipart/alternative; boundary=BOUND\r\n" +
		"\r\n" +
		"--BOUND\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"plain version\r\n" +
		"--BOUND\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		"<p>html version</p>\r\n" +
		"--BOUND--\r\n"

	m, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if m.Body != "plain version" {
		t.Errorf("Body = %q, want 'plain version' (plain preferred over html)", m.Body)
	}
}

func TestParseMessageHTMLFallback(t *testing.T) {
	raw := "From: a@b.com\r\n" +
		"Subject: hi\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		"<html><body><p>Hello</p><p>World</p></body></html>\r\n"

	m, err := ParseMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Body, "Hello") || !strings.Contains(m.Body, "World") {
		t.Errorf("Body = %q, want stripped HTML containing Hello/World", m.Body)
	}
	if strings.Contains(m.Body, "<p>") {
		t.Errorf("Body still contains tags: %q", m.Body)
	}
}
