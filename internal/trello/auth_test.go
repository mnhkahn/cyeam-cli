package trello

import (
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
	credentials, source, err := ResolveCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if credentials.APIKey != "key" || credentials.Token != "token" || source != "environment" {
		t.Fatalf("credentials = %#v, source = %q", credentials, source)
	}
}

func TestResolveCredentialsRejectsPartialEnvironment(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "key")
	t.Setenv("TRELLO_TOKEN", "")
	if _, _, err := ResolveCredentials(); err == nil {
		t.Fatal("expected partial environment to fail")
	}
}
