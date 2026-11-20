package trello

import (
	"context"
	"net/url"
	"testing"
)

func TestAuthorizationURLRequestsNonExpiringReadWriteToken(t *testing.T) {
	parsed, err := url.Parse(AuthorizationURL("abc123"))
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"key":           "abc123",
		"expiration":    "never",
		"scope":         "read,write",
		"response_type": "token",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestResolveCredentialsPrefersCompleteEnvironment(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", " key ")
	t.Setenv("TRELLO_TOKEN", " token ")
	session, err := ResolveSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.APIKey != "key" || session.Token != "token" || session.Source != "environment" {
		t.Fatalf("unexpected credentials or source %q", session.Source)
	}
}

func TestResolveCredentialsRejectsPartialEnvironment(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "key")
	t.Setenv("TRELLO_TOKEN", "")
	if _, err := ResolveSession(context.Background()); err == nil {
		t.Fatal("expected partial environment to fail")
	}
}
