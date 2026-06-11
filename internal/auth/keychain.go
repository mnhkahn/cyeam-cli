package auth

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keychainService = "cyeam-cli"

type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Expiry       int64  `json:"expiry"`
}

func StoreToken(t TokenSet) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return keyring.Set(keychainService, "microsoft_graph", string(data))
}

func LoadToken() (TokenSet, error) {
	data, err := keyring.Get(keychainService, "microsoft_graph")
	if err != nil {
		return TokenSet{}, fmt.Errorf("no token found, run `cyeam login` first")
	}
	var t TokenSet
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return TokenSet{}, fmt.Errorf("unmarshal token: %w", err)
	}
	return t, nil
}

func DeleteToken() error {
	return keyring.Delete(keychainService, "microsoft_graph")
}