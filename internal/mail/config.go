package mail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Account describes one mailbox. The password itself is never stored here; it
// is read at runtime from the environment variable named by PasswordEnv.
type Account struct {
	Name        string `json:"name"`
	IMAPHost    string `json:"imap_host"`
	IMAPPort    int    `json:"imap_port"`
	Username    string `json:"username"`
	PasswordEnv string `json:"password_env"`
	SMTPHost    string `json:"smtp_host,omitempty"`
	SMTPPort    int    `json:"smtp_port,omitempty"`
}

// Config is the on-disk ~/.cyeam/mail.json structure.
type Config struct {
	Accounts []Account `json:"accounts"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cyeam", "mail.json"), nil
}

// LoadConfig reads ~/.cyeam/mail.json.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no mail config at %s; create it with your accounts", path)
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// FindAccount returns the account with the given name.
func (c *Config) FindAccount(name string) (Account, error) {
	for _, a := range c.Accounts {
		if a.Name == name {
			return a, nil
		}
	}
	names := make([]string, len(c.Accounts))
	for i, a := range c.Accounts {
		names[i] = a.Name
	}
	return Account{}, fmt.Errorf("account %q not found (configured: %s)", name, strings.Join(names, ", "))
}

// Password reads the account's app-specific password from its environment
// variable.
func (a Account) Password() (string, error) {
	if a.PasswordEnv == "" {
		return "", fmt.Errorf("account %q has no password_env configured", a.Name)
	}
	pass := os.Getenv(a.PasswordEnv)
	if pass == "" {
		return "", fmt.Errorf("environment variable %s is not set (app-specific password for %q)", a.PasswordEnv, a.Name)
	}
	return pass, nil
}

// IMAPAddr returns "host:port", defaulting the port to 993.
func (a Account) IMAPAddr() string {
	port := a.IMAPPort
	if port == 0 {
		port = 993
	}
	return fmt.Sprintf("%s:%d", a.IMAPHost, port)
}

// SMTPAddr returns "host:port" for sending. When smtp_host is omitted it is
// derived from the IMAP host (imap.* -> smtp.*); the port defaults to 465.
func (a Account) SMTPAddr() (host, addr string) {
	host = a.SMTPHost
	if host == "" {
		host = strings.Replace(a.IMAPHost, "imap.", "smtp.", 1)
	}
	port := a.SMTPPort
	if port == 0 {
		port = 465
	}
	return host, fmt.Sprintf("%s:%d", host, port)
}
