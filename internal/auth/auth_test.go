package auth

import (
	"strings"
	"testing"
	"time"
)

func TestScopesRequestOfflineAccess(t *testing.T) {
	if !strings.Contains(scopes, "offline_access") {
		t.Fatalf("scopes = %q, want offline_access for refresh tokens", scopes)
	}
}

func TestMergeRefreshedTokenKeepsExistingRefreshTokenWhenOmitted(t *testing.T) {
	now := time.Unix(1000, 0)
	current := TokenSet{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		Expiry:       1000,
	}
	refreshed := tokenResponse{
		AccessToken: "new-access",
		ExpiresIn:   3600,
	}

	got := mergeRefreshedToken(current, refreshed, now)

	if got.AccessToken != "new-access" {
		t.Fatalf("access token = %q", got.AccessToken)
	}
	if got.RefreshToken != "old-refresh" {
		t.Fatalf("refresh token = %q, want old refresh token", got.RefreshToken)
	}
	if got.Expiry != 4600 {
		t.Fatalf("expiry = %d, want 4600", got.Expiry)
	}
}

func TestMergeRefreshedTokenUsesRotatedRefreshToken(t *testing.T) {
	now := time.Unix(1000, 0)
	current := TokenSet{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		Expiry:       1000,
	}
	refreshed := tokenResponse{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresIn:    3600,
	}

	got := mergeRefreshedToken(current, refreshed, now)

	if got.RefreshToken != "new-refresh" {
		t.Fatalf("refresh token = %q, want rotated refresh token", got.RefreshToken)
	}
}
