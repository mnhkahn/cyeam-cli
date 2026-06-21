package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const keychainService = "cyeam-cli"

type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Expiry       int64  `json:"expiry"`
}

func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cyeam")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

func StoreToken(t TokenSet) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyring.Set(keychainService, "microsoft_graph", string(data)); err == nil {
		removeTokenFile()
		return nil
	}
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func removeTokenFile() {
	path, err := tokenFilePath()
	if err == nil {
		os.Remove(path)
	}
}

func LoadToken() (TokenSet, error) {
	s, err := keyring.Get(keychainService, "microsoft_graph")
	if err == nil {
		var t TokenSet
		if err := json.Unmarshal([]byte(s), &t); err != nil {
			return TokenSet{}, fmt.Errorf("unmarshal token: %w", err)
		}
		return t, nil
	}
	path, ferr := tokenFilePath()
	if ferr != nil {
		return TokenSet{}, fmt.Errorf("no token found, run `cyeam login` first")
	}
	b, ferr := os.ReadFile(path)
	if ferr != nil {
		return TokenSet{}, fmt.Errorf("no token found, run `cyeam login` first")
	}
	var t TokenSet
	if err := json.Unmarshal(b, &t); err != nil {
		return TokenSet{}, fmt.Errorf("unmarshal token: %w", err)
	}
	return t, nil
}

func DeleteToken() error {
	removeTokenFile()
	keyring.Delete(keychainService, "microsoft_graph")
	return nil
}
