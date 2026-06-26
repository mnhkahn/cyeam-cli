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
	user, err := acc.GetUsername()
	if err != nil || user != "a@cyeam.com" || acc.IMAPAddr() != "imap.zoho.com:993" {
		t.Errorf("unexpected account: user=%q, err=%v, addr=%s", user, err, acc.IMAPAddr())
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

func TestGetUsername(t *testing.T) {
	// username 明文配置可继续工作
	acc1 := Account{Name: "cyeam", Username: "a@cyeam.com"}
	user, err := acc1.GetUsername()
	if err != nil || user != "a@cyeam.com" {
		t.Errorf("plain username: got %q, %v; want a@cyeam.com", user, err)
	}

	// username_env 配置可从 env 读取
	t.Setenv("TEST_USERNAME", "env@cyeam.com")
	acc2 := Account{Name: "cyeam", UsernameEnv: "TEST_USERNAME"}
	user, err = acc2.GetUsername()
	if err != nil || user != "env@cyeam.com" {
		t.Errorf("username_env: got %q, %v; want env@cyeam.com", user, err)
	}

	// username 优先级高于 username_env
	t.Setenv("TEST_USERNAME2", "env@cyeam.com")
	acc3 := Account{Name: "cyeam", Username: "explicit@cyeam.com", UsernameEnv: "TEST_USERNAME2"}
	user, err = acc3.GetUsername()
	if err != nil || user != "explicit@cyeam.com" {
		t.Errorf("username priority: got %q, %v; want explicit@cyeam.com", user, err)
	}

	// 缺少 username 和 username_env 时返回错误
	acc4 := Account{Name: "cyeam"}
	_, err = acc4.GetUsername()
	if err == nil {
		t.Error("expected error when neither username nor username_env set")
	}
	if err.Error() != `account "cyeam" has no username or username_env configured` {
		t.Errorf("unexpected error message: %v", err)
	}

	// username_env 指向的 env 未设置时返回错误
	acc5 := Account{Name: "cyeam", UsernameEnv: "UNSET_VAR"}
	_, err = acc5.GetUsername()
	if err == nil {
		t.Error("expected error when username_env var unset")
	}
	if err.Error() != `environment variable UNSET_VAR is not set (username for "cyeam")` {
		t.Errorf("unexpected error message: %v", err)
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
