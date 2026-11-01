package paddleocr

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func mockCredentials(t *testing.T) map[string]string {
	t.Helper()
	oldSet, oldGet, oldDelete, oldHome := keyringSet, keyringGet, keyringDelete, userHomeDir
	dir := t.TempDir()
	userHomeDir = func() (string, error) { return dir, nil }
	values := map[string]string{}
	keyringSet = func(s, a, v string) error { values[s+"/"+a] = v; return nil }
	keyringGet = func(s, a string) (string, error) {
		v, ok := values[s+"/"+a]
		if !ok {
			return "", keyring.ErrNotFound
		}
		return v, nil
	}
	keyringDelete = func(s, a string) error { delete(values, s+"/"+a); return nil }
	t.Cleanup(func() { keyringSet, keyringGet, keyringDelete, userHomeDir = oldSet, oldGet, oldDelete, oldHome })
	return values
}
func TestSavedCredentialsAndEnvironmentIsolation(t *testing.T) {
	mockCredentials(t)
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	c := Credentials{URL: "https://example.invalid/ocr", Token: "private-token"}
	storage, err := StoreCredentials(c)
	if err != nil || storage != "keyring" {
		t.Fatal(storage, err)
	}
	client, err := Config("")
	if err != nil || client.URL != c.URL || client.Token != c.Token {
		t.Fatal("saved config not loaded", err)
	}
	t.Setenv("OCR_API_URL", "https://other.invalid/ocr")
	if _, err := Config(""); err == nil {
		t.Fatal("saved token was reused with another endpoint")
	}
	t.Setenv("OCR_API_TOKEN", "override")
	client, err = Config("")
	if err != nil || client.Token != "override" {
		t.Fatal(err)
	}
	if err := DeleteCredentials(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadCredentials(); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
}
func TestFileFallbackUpdatesAndPermissions(t *testing.T) {
	mockCredentials(t)
	first := Credentials{URL: "https://example.invalid/ocr", Token: "old"}
	if _, err := StoreCredentials(first); err != nil {
		t.Fatal(err)
	}
	keyringSet = func(string, string, string) error { return errors.New("unavailable") }
	replacement := Credentials{URL: first.URL, Token: "new"}
	storage, err := StoreCredentials(replacement)
	if err != nil || storage != "file" {
		t.Fatal(storage, err)
	}
	got, source, err := LoadCredentials()
	if err != nil || got != replacement || source != "file" {
		t.Fatal("old keyring shadowed new file", err)
	}
	path, _ := credentialsPath()
	stat, err := os.Stat(path)
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("credential permissions", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := StoreCredentials(replacement); err != nil {
		t.Fatal(err)
	}
	stat, _ = os.Stat(path)
	if stat.Mode().Perm() != 0600 {
		t.Fatal("replacement did not fix permissions")
	}
	keyringDelete = func(string, string) error { return errors.New("locked") }
	if err := DeleteCredentials(); err == nil {
		t.Fatal("clear claimed success while old keyring entry remained")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed clear reactivated old keyring entry")
	}
	keyringDelete = func(string, string) error { return nil }
	if err := DeleteCredentials(); err != nil {
		t.Fatal(err)
	}
}
func TestStoredConfigMalformedDoesNotFallBackToOldToken(t *testing.T) {
	mockCredentials(t)
	if _, err := StoreCredentials(Credentials{URL: "https://example.invalid", Token: "old"}); err != nil {
		t.Fatal(err)
	}
	keyringSet = func(string, string, string) error { return errors.New("unavailable") }
	if _, err := StoreCredentials(Credentials{URL: "https://example.invalid", Token: "new"}); err != nil {
		t.Fatal(err)
	}
	path, _ := credentialsPath()
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadCredentials(); err == nil {
		t.Fatal("malformed file silently fell back")
	}
	for _, c := range []Credentials{{}, {URL: "https://u:secret@example.invalid", Token: "secret"}, {URL: "https://example.invalid", Token: "one\ntwo"}} {
		if _, err := StoreCredentials(c); err == nil {
			t.Fatal("invalid credentials accepted")
		}
	}
}

func TestMissingCredentialsDisableOCR(t *testing.T) {
	mockCredentials(t)
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	if _, err := Config(""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("missing credentials must disable image checking: %v", err)
	}
}

func TestEndpointOnlyDoesNotEnableOCRorReuseToken(t *testing.T) {
	mockCredentials(t)
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	if err := StoreEndpoint("https://new.invalid/ocr"); err != nil {
		t.Fatal(err)
	}
	if endpoint, err := LoadEndpoint(); err != nil || endpoint != "https://new.invalid/ocr" {
		t.Fatal(err)
	}
	if _, err := Config(""); !errors.Is(err, ErrNotConfigured) {
		t.Fatal("endpoint alone enabled OCR", err)
	}
	original := Credentials{URL: "https://old.invalid/ocr", Token: "existing"}
	if _, err := StoreCredentials(original); err != nil {
		t.Fatal(err)
	}
	if err := StoreEndpoint("https://new.invalid/ocr"); err != nil {
		t.Fatal(err)
	}
	got, _, err := LoadCredentials()
	if err != nil || got != original {
		t.Fatal("endpoint-only changed active credentials", err)
	}
	if err := DeleteCredentials(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEndpoint(); !errors.Is(err, ErrNotConfigured) {
		t.Fatal("clear kept endpoint", err)
	}
}
