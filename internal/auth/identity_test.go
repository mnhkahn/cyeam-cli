package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/coreos/go-oidc/v3/oidc"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLoginIdentityVerifiesMicrosoftSignatureAndClient(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	enc := base64.RawURLEncoding.EncodeToString
	jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256", "n": enc(key.N.Bytes()), "e": "AQAB"}}})
	transport := resourceTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(jwks)))}, nil
	})
	ctx := oidc.ClientContext(context.Background(), &http.Client{Transport: transport})
	sign := func(audience, issuer string) string {
		claims, _ := json.Marshal(map[string]any{"iss": issuer, "aud": audience, "sub": "same-account", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()})
		body := enc([]byte(`{"alg":"RS256","kid":"test"}`)) + "." + enc(claims)
		digest := sha256.Sum256([]byte(body))
		signature, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if e != nil {
			t.Fatal(e)
		}
		return body + "." + enc(signature)
	}
	identity, e := accountIdentity(ctx, sign(clientID, consumerIssuer))
	if e != nil || identity != consumerIssuer+"|same-account" {
		t.Fatalf("identity=%q error=%v", identity, e)
	}
	if _, e = accountIdentity(ctx, sign("another-client", consumerIssuer)); e == nil {
		t.Fatal("accepted ID token for a different client")
	}
	if _, e = accountIdentity(ctx, sign(clientID, "https://invalid.example")); e == nil {
		t.Fatal("accepted foreign issuer")
	}
}
