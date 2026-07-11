package trello

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "key", Token: "token"}
}

func TestCreateCardEncodesAuthAndFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cards" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("key") != "key" || q.Get("token") != "token" || q.Get("idList") != "list-1" || q.Get("name") != "数学作业" {
			t.Fatalf("unexpected query: %v", q)
		}
		_, _ = io.WriteString(w, `{"id":"card-1"}`)
	})
	if _, err := client.CreateCard(context.Background(), url.Values{"idList": {"list-1"}, "name": {"数学作业"}}); err != nil {
		t.Fatal(err)
	}
}

func TestAttachFileUsesMultipart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "proof.jpg")
	if err := os.WriteFile(file, []byte("photo"), 0600); err != nil {
		t.Fatal(err)
	}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cards/card-1/attachments" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		body, _ := io.ReadAll(f)
		if string(body) != "photo" {
			t.Fatalf("attachment = %q", body)
		}
		if r.URL.Query().Get("key") != "key" {
			t.Fatal("missing key")
		}
		_, _ = io.WriteString(w, `{}`)
	})
	if _, err := client.AttachFile(context.Background(), "card-1", file, ""); err != nil {
		t.Fatal(err)
	}
}

func TestRequestIncludesAPIError(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "invalid token", http.StatusUnauthorized) })
	_, err := client.Boards(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("error = %v", err)
	}
}
