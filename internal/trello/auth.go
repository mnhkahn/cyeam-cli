package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keychainService = "cyeam-cli"
	keychainAccount = "trello"
)

type Credentials struct {
	APIKey string `json:"api_key"`
	Token  string `json:"token"`
}

type Session struct {
	Credentials
	Source    string
	BoardID   string
	BoardName string
}

func ResolveSession(ctx context.Context) (Session, error) {
	return resolveSession(ctx, loadCNoteConnection, LoadCredentials)
}

func resolveSession(ctx context.Context, cloud func(context.Context) (Session, error), local func() (Credentials, error)) (Session, error) {
	key, token := strings.TrimSpace(os.Getenv("TRELLO_API_KEY")), strings.TrimSpace(os.Getenv("TRELLO_TOKEN"))
	if key != "" || token != "" {
		if key == "" || token == "" {
			return Session{}, fmt.Errorf("set both TRELLO_API_KEY and TRELLO_TOKEN, or unset both to use saved credentials")
		}
		return Session{Credentials: Credentials{APIKey: key, Token: token}, Source: "environment"}, nil
	}
	session, err := cloud(ctx)
	if err != nil || session.Source != "" {
		return session, err
	}
	credentials, err := local()
	if err != nil {
		return Session{}, err
	}
	return Session{Credentials: credentials, Source: "stored"}, nil
}

func StoreCredentials(credentials Credentials) error {
	credentials.APIKey = strings.TrimSpace(credentials.APIKey)
	credentials.Token = strings.TrimSpace(credentials.Token)
	if credentials.APIKey == "" || credentials.Token == "" {
		return fmt.Errorf("API key and token are required")
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	if err := keyring.Set(keychainService, keychainAccount, string(data)); err == nil {
		removeCredentialsFile()
		return nil
	}
	path, err := credentialsFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func LoadCredentials() (Credentials, error) {
	if value, err := keyring.Get(keychainService, keychainAccount); err == nil {
		return decodeCredentials([]byte(value))
	}
	path, err := credentialsFilePath()
	if err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("not logged in to Trello; use `cyeam login` with a saved CNote connection or run `cyeam trello login --key <api-key>`")
	}
	return decodeCredentials(data)
}

func DeleteCredentials() error {
	removeCredentialsFile()
	_ = keyring.Delete(keychainService, keychainAccount)
	return nil
}

func AuthorizationURL(apiKey string) string {
	values := url.Values{
		"expiration":    {"never"},
		"name":          {"cyeam"},
		"scope":         {"read,write"},
		"response_type": {"token"},
		"key":           {strings.TrimSpace(apiKey)},
	}
	return "https://trello.com/1/authorize?" + values.Encode()
}

func credentialsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cyeam")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "trello.json"), nil
}

func removeCredentialsFile() {
	if path, err := credentialsFilePath(); err == nil {
		_ = os.Remove(path)
	}
}

func decodeCredentials(data []byte) (Credentials, error) {
	var credentials Credentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return Credentials{}, fmt.Errorf("decode Trello credentials: %w", err)
	}
	if credentials.APIKey == "" || credentials.Token == "" {
		return Credentials{}, fmt.Errorf("stored Trello credentials are incomplete; run `cyeam trello login --key <api-key>`")
	}
	return credentials, nil
}
