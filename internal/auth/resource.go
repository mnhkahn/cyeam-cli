package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/zalando/go-keyring"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var resourceMu sync.Mutex

func sessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func resourceName(scope string) string {
	h := sha256.Sum256([]byte(scope))
	return "resource-" + hex.EncodeToString(h[:])
}
func resourcePath(name string) (string, error) {
	p, e := tokenFilePath()
	if e != nil {
		return "", e
	}
	return filepath.Join(filepath.Dir(p), name+".json"), nil
}
func loadResource(scope string) (TokenSet, error) {
	name := resourceName(scope)
	if raw, e := keyring.Get(keychainService, name); e == nil {
		var t TokenSet
		e = json.Unmarshal([]byte(raw), &t)
		return t, e
	}
	p, e := resourcePath(name)
	if e != nil {
		return TokenSet{}, e
	}
	raw, e := os.ReadFile(p)
	if e != nil {
		return TokenSet{}, e
	}
	var t TokenSet
	e = json.Unmarshal(raw, &t)
	return t, e
}
func storeResource(scope string, t TokenSet) error {
	raw, e := json.Marshal(t)
	if e != nil {
		return e
	}
	p, e := resourcePath(resourceName(scope))
	if e != nil {
		return e
	}
	if keyring.Set(keychainService, resourceName(scope), string(raw)) == nil {
		return os.WriteFile(p, []byte("{}"), 0600)
	}
	return os.WriteFile(p, raw, 0600)
}
func DeleteResourceTokens() error {
	p, e := tokenFilePath()
	if e != nil {
		return e
	}
	paths, e := filepath.Glob(filepath.Join(filepath.Dir(p), "resource-*.json"))
	if e != nil {
		return e
	}
	for _, p := range paths {
		keyring.Delete(keychainService, strings.TrimSuffix(filepath.Base(p), ".json"))
		if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	return nil
}

// Gateway access tokens are isolated from Graph tokens. Refresh tokens stay in the CLI.
func GetResourceAccessToken(ctx context.Context, scope string) (string, error) {
	resourceMu.Lock()
	defer resourceMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	current, e := LoadToken()
	if e != nil {
		return "", e
	}
	if current.SessionID == "" {
		current.SessionID = sessionID()
		if e = StoreToken(current); e != nil {
			return "", e
		}
	}
	if cached, e := loadResource(scope); e == nil && cached.SessionID == current.SessionID && cached.AccessToken != "" && cached.Expiry > time.Now().Unix()+60 {
		return cached.AccessToken, nil
	}
	if current.RefreshToken == "" {
		return "", fmt.Errorf("gateway permission is missing; run `cyeam login` or `cyeam device login`")
	}
	fresh, e := refreshTokenWithScope(ctx, current.RefreshToken, scope)
	if e != nil {
		return "", fmt.Errorf("gateway authorization required; run `cyeam device login`: %w", e)
	}
	if e = saveResource(current, scope, fresh); e != nil {
		return "", e
	}
	return fresh.AccessToken, nil
}
func saveResource(current TokenSet, scope string, fresh tokenResponse) error {
	latest, e := LoadToken()
	if e != nil {
		return e
	}
	if latest.SessionID != current.SessionID {
		return fmt.Errorf("account changed during login; retry with the current account")
	}
	if fresh.RefreshToken != "" {
		latest.RefreshToken = fresh.RefreshToken
		if e = StoreToken(latest); e != nil {
			return e
		}
	}
	return storeResource(scope, TokenSet{AccessToken: fresh.AccessToken, Expiry: time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second).Unix(), SessionID: current.SessionID})
}
func LoginResource(ctx context.Context, scope string, stdout, stderr io.Writer) error {
	if _, e := GetResourceAccessToken(ctx, scope); e == nil {
		return nil
	}
	current, e := LoadToken()
	if e != nil {
		return e
	}
	if current.AccountID == "" {
		return fmt.Errorf("run cyeam login again to verify the Microsoft account before granting gateway access")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	dc, e := requestDeviceCodeWithScope(ctx, scope+" offline_access openid profile")
	if e != nil {
		return e
	}
	fmt.Fprintf(stdout, "Gateway authorization: %s\nCode: %s\n", dc.VerificationURI, dc.UserCode)
	_ = OpenBrowser(dc.VerificationURI)
	fresh, e := pollForToken(ctx, dc.DeviceCode, dc.Interval)
	if e != nil {
		return e
	}
	account, err := accountIdentity(ctx, fresh.IDToken)
	if err != nil {
		return err
	}
	if account != current.AccountID {
		return fmt.Errorf("gateway authorization used a different Microsoft account; retry with the account used by cyeam login")
	}
	return saveResource(current, scope, fresh)
}
