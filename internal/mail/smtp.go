package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"
)

// Send delivers a UTF-8 message via the account's SMTP server. Port 465 uses
// implicit TLS; any other port uses STARTTLS.
func Send(acc Account, to, cc []string, subject, body string) error {
	user, err := acc.GetUsername()
	if err != nil {
		return err
	}
	pass, err := acc.Password()
	if err != nil {
		return err
	}
	host, addr := acc.SMTPAddr()
	auth := smtp.PlainAuth("", user, pass, host)

	msg := buildMIME(user, to, cc, subject, body)
	rcpts := append(append([]string{}, to...), cc...)
	if len(rcpts) == 0 {
		return fmt.Errorf("no recipients")
	}

	port := acc.SMTPPort
	if port == 0 {
		port = 465
	}
	if port == 465 {
		return sendTLS(host, addr, auth, user, rcpts, msg)
	}
	return sendSTARTTLS(addr, auth, user, rcpts, msg)
}

func sendTLS(host, addr string, auth smtp.Auth, from string, rcpts []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Quit()
	return deliver(c, auth, from, rcpts, msg)
}

func sendSTARTTLS(addr string, auth smtp.Auth, from string, rcpts []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	defer c.Quit()
	host := addr[:strings.LastIndex(addr, ":")]
	if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	return deliver(c, auth, from, rcpts, msg)
}

func deliver(c *smtp.Client, auth smtp.Auth, from string, rcpts []string, msg []byte) error {
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, r := range rcpts {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("rcpt %s: %w", r, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func buildMIME(from string, to, cc []string, subject, body string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(to, ", "))
	if len(cc) > 0 {
		fmt.Fprintf(&b, "Cc: %s\r\n", strings.Join(cc, ", "))
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return b.Bytes()
}
