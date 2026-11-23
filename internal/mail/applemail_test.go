package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppleMailListsAndReadsOnlyCompleteInboxMessages(t *testing.T) {
	dir := t.TempDir()
	write := func(name, date, subject string) {
		t.Helper()
		raw := fmt.Sprintf("From: Sender <sender@example.test>\r\nSubject: %s\r\nDate: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nMessage body", subject, date)
		file := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(fmt.Sprintf("%d\n%s<?xml trailing Apple metadata?>", len(raw), raw)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("account-a/INBOX.mbox/Data/1/Messages/2.emlx", "Mon, 23 Nov 2026 10:00:00 +0000", "Older")
	write("account-b/收件箱.mbox/Data/Messages/2.emlx", "Tue, 24 Nov 2026 10:00:00 +0000", "Newer")
	write("account-a/INBOX.mbox/Data/Messages/3.partial.emlx", "Wed, 25 Nov 2026 10:00:00 +0000", "Incomplete")
	write("account-a/INBOX.mbox/Archive.mbox/Data/Messages/4.emlx", "Wed, 25 Nov 2026 10:00:00 +0000", "Nested archive")
	write("account-a/Sent.mbox/Data/Messages/5.emlx", "Wed, 25 Nov 2026 10:00:00 +0000", "Sent")

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	messages, err := ListAppleMail(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Subject != "Newer" || messages[1].Subject != "Older" ||
		messages[0].ID == messages[1].ID {
		t.Fatalf("unexpected summaries: %+v", messages)
	}

	msg, err := ReadAppleMail(root, messages[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "Newer" || msg.Body != "Message body" {
		t.Fatalf("unexpected cached message: %+v", msg)
	}
	if _, err := ReadAppleMail(root, "../outside"); err == nil {
		t.Fatal("expected a path to be rejected as an ID")
	}
	if _, err := ReadAppleMail(root, strings.Repeat("0", 64)); err == nil {
		t.Fatal("expected a missing message to return an error")
	}
}
