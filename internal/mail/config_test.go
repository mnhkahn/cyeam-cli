package mail

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAndFindAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cyeamDir := filepath.Join(dir, ".cyeam")
	if err := os.MkdirAll(cyeamDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"accounts":[{"name":"cyeam","imap_host":"imap.zoho.com","imap_port":993,"username":"a@cyeam.com","password_env":"ZOHO_MAIL_PASS"}]}`
	if err := os.WriteFile(filepath.Join(cyeamDir, "mail.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	acc, err := loaded.FindAccount("cyeam")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Username != "a@cyeam.com" || acc.IMAPAddr() != "imap.zoho.com:993" {
		t.Errorf("unexpected account: %+v", acc)
	}
	if _, err := loaded.FindAccount("nope"); err == nil {
		t.Error("expected error for missing account")
	}
}

func TestPasswordFromEnv(t *testing.T) {
	acc := Account{Name: "cyeam", PasswordEnv: "TEST_MAIL_PASS"}
	if _, err := acc.Password(); err == nil {
		t.Error("expected error when env var unset")
	}
	t.Setenv("TEST_MAIL_PASS", "secret")
	got, err := acc.Password()
	if err != nil || got != "secret" {
		t.Errorf("Password() = %q, %v; want secret", got, err)
	}
}

func TestSMTPAddrDerivation(t *testing.T) {
	// Derived from IMAP host when smtp_host omitted.
	acc := Account{IMAPHost: "imap.zoho.com"}
	host, addr := acc.SMTPAddr()
	if host != "smtp.zoho.com" || addr != "smtp.zoho.com:465" {
		t.Errorf("derived SMTP = %s / %s, want smtp.zoho.com:465", host, addr)
	}
	// Explicit overrides win.
	acc2 := Account{IMAPHost: "imap.x.com", SMTPHost: "mail.x.com", SMTPPort: 587}
	host2, addr2 := acc2.SMTPAddr()
	if host2 != "mail.x.com" || addr2 != "mail.x.com:587" {
		t.Errorf("explicit SMTP = %s / %s, want mail.x.com:587", host2, addr2)
	}
}
