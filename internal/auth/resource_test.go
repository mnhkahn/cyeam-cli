package auth

import (
	"context"
	"github.com/zalando/go-keyring"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type resourceTransport func(*http.Request) (*http.Response, error)

func (f resourceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestResourceRefreshPreservesGraphAccessToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("CYEAM_AUTH_DIR", t.TempDir())
	original := TokenSet{AccessToken: "graph-access", RefreshToken: "original-refresh", Expiry: time.Now().Add(time.Hour).Unix(), SessionID: "account-a"}
	if e := StoreToken(original); e != nil {
		t.Fatal(e)
	}
	old := http.DefaultClient
	defer func() { http.DefaultClient = old }()
	calls := 0
	http.DefaultClient = &http.Client{Transport: resourceTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if e := r.ParseForm(); e != nil {
			t.Fatal(e)
		}
		if r.Form.Get("scope") != "api://gateway/Device.Control" {
			t.Fatalf("wrong scope: %v", r.Form)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"gateway-access","refresh_token":"rotated-refresh","expires_in":3600}`)), Header: make(http.Header)}, nil
	})}
	for i := 0; i < 2; i++ {
		token, e := GetResourceAccessToken(context.Background(), "api://gateway/Device.Control")
		if e != nil || token != "gateway-access" {
			t.Fatalf("token=%q err=%v", token, e)
		}
	}
	if calls != 1 {
		t.Fatalf("cache missed: %d requests", calls)
	}
	graph, e := LoadToken()
	if e != nil {
		t.Fatal(e)
	}
	if graph.AccessToken != "graph-access" || graph.RefreshToken != "rotated-refresh" || graph.SessionID != "account-a" {
		t.Fatalf("Graph credentials were replaced: %+v", graph)
	}
	// A different account cannot reuse the cached token.
	graph.SessionID = "account-b"
	_ = StoreToken(graph)
	if _, e = GetResourceAccessToken(context.Background(), "api://gateway/Device.Control"); e != nil {
		t.Fatal(e)
	}
	if calls != 2 {
		t.Fatal("cross-account cache reused")
	}
	if e = Logout(); e != nil {
		t.Fatal(e)
	}
	if _, e = LoadToken(); e == nil {
		t.Fatal("logout retained Graph credentials")
	}
}
