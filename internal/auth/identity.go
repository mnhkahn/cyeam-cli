package auth

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
)

const consumerIssuer = "https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"

// ID tokens identify the CLI account; only gateway-audience access tokens are sent to the gateway.
func accountIdentity(ctx context.Context, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("Microsoft did not return an ID token; run cyeam login again")
	}
	keys := oidc.NewRemoteKeySet(ctx, "https://login.microsoftonline.com/common/discovery/v2.0/keys")
	verifier := oidc.NewVerifier(consumerIssuer, keys, &oidc.Config{ClientID: clientID, SupportedSigningAlgs: []string{"RS256"}})
	token, e := verifier.Verify(ctx, raw)
	if e != nil {
		return "", fmt.Errorf("verify Microsoft account: %w", e)
	}
	if token.Subject == "" {
		return "", fmt.Errorf("Microsoft returned no account identity")
	}
	return token.Issuer + "|" + token.Subject, nil
}
