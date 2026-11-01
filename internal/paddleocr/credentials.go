package paddleocr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const credentialService = "cyeam-cli"
const credentialAccount = "paddleocr"

var ErrNotConfigured = errors.New("PaddleOCR image checking is unavailable without your own endpoint and token; run `cyeam ocr config set --url <endpoint>`")
var userHomeDir = os.UserHomeDir
var keyringSet = keyring.Set
var keyringGet = keyring.Get
var keyringDelete = keyring.Delete

type Credentials struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func validateCredentials(c Credentials) error {
	if strings.TrimSpace(c.URL) == "" || strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("OCR URL and token are required")
	}
	u, err := url.Parse(c.URL)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("OCR URL must be an HTTP(S) endpoint without user information or fragment")
	}
	if strings.ContainsAny(c.Token, "\r\n") {
		return fmt.Errorf("OCR token must be a single line")
	}
	return nil
}
func credentialsPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate OCR config directory failed")
	}
	return filepath.Join(home, ".cyeam", "paddleocr.json"), nil
}
func decodeCredentials(data []byte) (Credentials, error) {
	var c Credentials
	if json.Unmarshal(data, &c) != nil || validateCredentials(c) != nil {
		return Credentials{}, fmt.Errorf("stored OCR configuration is invalid; run `cyeam ocr config set --url <endpoint>` again")
	}
	return c, nil
}

// StoreCredentials saves a complete URL/token pair. A file takes precedence over
// the keyring, so a fallback update cannot be shadowed by an older keyring entry.
func StoreCredentials(c Credentials) (string, error) {
	c.URL = strings.TrimSpace(c.URL)
	c.Token = strings.TrimSpace(c.Token)
	if err := validateCredentials(c); err != nil {
		return "", err
	}
	path, err := credentialsPath()
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(c)
	if err := keyringSet(credentialService, credentialAccount, string(data)); err == nil {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("remove previous OCR credential file failed")
		}
		return "keyring", nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create OCR config directory failed")
	}
	// Atomic replacement also gives an existing permissive file mode 0600.
	f, err := os.CreateTemp(dir, ".paddleocr-*")
	if err != nil {
		return "", fmt.Errorf("create OCR credential file failed")
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return "", fmt.Errorf("write OCR credential file failed")
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("write OCR credential file failed")
	}
	if err = os.Rename(temp, path); err != nil {
		return "", fmt.Errorf("replace OCR credential file failed")
	}
	return "file", nil
}
func LoadCredentials() (Credentials, string, error) {
	path, err := credentialsPath()
	if err != nil {
		return Credentials{}, "", err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		c, err := decodeCredentials(data)
		return c, "file", err
	}
	if !os.IsNotExist(err) {
		return Credentials{}, "", fmt.Errorf("read stored OCR credential file failed")
	}
	value, err := keyringGet(credentialService, credentialAccount)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) || errors.Is(err, keyring.ErrUnsupportedPlatform) {
			return Credentials{}, "", ErrNotConfigured
		}
		return Credentials{}, "", fmt.Errorf("read OCR credentials from system keyring failed")
	}
	c, err := decodeCredentials([]byte(value))
	return c, "keyring", err
}
func DeleteCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	// Clear the older keyring entry first, so deleting an overriding file never
	// accidentally reactivates old credentials if the keyring is locked.
	if err := keyringDelete(credentialService, credentialAccount); err != nil && !(errors.Is(err, keyring.ErrNotFound) || errors.Is(err, keyring.ErrUnsupportedPlatform)) {
		return fmt.Errorf("delete OCR credentials from system keyring failed")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete OCR credential file failed")
	}
	endpoint, err := endpointPath()
	if err != nil {
		return err
	}
	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete saved OCR endpoint failed")
	}
	return nil
}

func endpointPath() (string, error) {
	path, err := credentialsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "paddleocr-endpoint.json"), nil
}

// StoreEndpoint saves a setup preference only; it never changes an existing
// URL/token pair or sends a stored token to a newly selected endpoint.
func StoreEndpoint(endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if err := validateCredentials(Credentials{URL: endpoint, Token: "validation-only"}); err != nil {
		return err
	}
	path, err := endpointPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create OCR endpoint directory failed")
	}
	data, _ := json.Marshal(struct {
		URL string `json:"url"`
	}{endpoint})
	f, err := os.CreateTemp(filepath.Dir(path), ".paddleocr-endpoint-*")
	if err != nil {
		return fmt.Errorf("create OCR endpoint file failed")
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write OCR endpoint failed")
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("write OCR endpoint failed")
	}
	if err = os.Rename(temp, path); err != nil {
		return fmt.Errorf("save OCR endpoint failed")
	}
	return nil
}
func LoadEndpoint() (string, error) {
	path, err := endpointPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotConfigured
		}
		return "", fmt.Errorf("read OCR endpoint failed")
	}
	var value struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(data, &value) != nil || validateCredentials(Credentials{URL: value.URL, Token: "validation-only"}) != nil {
		return "", fmt.Errorf("saved OCR endpoint is invalid")
	}
	return value.URL, nil
}
