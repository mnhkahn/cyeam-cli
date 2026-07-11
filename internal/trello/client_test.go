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

func TestBoardCRUDEndpoints(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantQuery  map[string]string
		call       func(c *Client) error
	}{
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/boards/b1",
			call: func(c *Client) error { _, err := c.Board(context.Background(), "b1"); return err },
		},
		{
			name: "create", wantMethod: http.MethodPost, wantPath: "/boards/",
			wantQuery: map[string]string{"name": "看板"},
			call: func(c *Client) error {
				_, err := c.CreateBoard(context.Background(), url.Values{"name": {"看板"}})
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPut, wantPath: "/boards/b1",
			wantQuery: map[string]string{"name": "新名字"},
			call: func(c *Client) error {
				_, err := c.UpdateBoard(context.Background(), "b1", url.Values{"name": {"新名字"}})
				return err
			},
		},
		{
			name: "delete", wantMethod: http.MethodDelete, wantPath: "/boards/b1",
			call: func(c *Client) error { _, err := c.DeleteBoard(context.Background(), "b1"); return err },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.wantMethod || r.URL.Path != tc.wantPath {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				q := r.URL.Query()
				if q.Get("key") != "key" || q.Get("token") != "token" {
					t.Fatalf("missing auth: %v", q)
				}
				for k, v := range tc.wantQuery {
					if q.Get(k) != v {
						t.Fatalf("query %q = %q, want %q", k, q.Get(k), v)
					}
				}
				_, _ = io.WriteString(w, `{}`)
			})
			if err := tc.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestListCRUDEndpoints(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantQuery  map[string]string
		call       func(c *Client) error
	}{
		{
			name: "get", wantMethod: http.MethodGet, wantPath: "/lists/l1",
			call: func(c *Client) error { _, err := c.List(context.Background(), "l1"); return err },
		},
		{
			name: "create", wantMethod: http.MethodPost, wantPath: "/lists",
			wantQuery: map[string]string{"idBoard": "b1", "name": "待执行"},
			call: func(c *Client) error {
				_, err := c.CreateList(context.Background(), url.Values{"idBoard": {"b1"}, "name": {"待执行"}})
				return err
			},
		},
		{
			name: "update", wantMethod: http.MethodPut, wantPath: "/lists/l1",
			wantQuery: map[string]string{"name": "已提交"},
			call: func(c *Client) error {
				_, err := c.UpdateList(context.Background(), "l1", url.Values{"name": {"已提交"}})
				return err
			},
		},
		{
			name: "archive", wantMethod: http.MethodPut, wantPath: "/lists/l1/closed",
			wantQuery: map[string]string{"value": "true"},
			call: func(c *Client) error { _, err := c.ArchiveList(context.Background(), "l1"); return err },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.wantMethod || r.URL.Path != tc.wantPath {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				q := r.URL.Query()
				for k, v := range tc.wantQuery {
					if q.Get(k) != v {
						t.Fatalf("query %q = %q, want %q", k, q.Get(k), v)
					}
				}
				_, _ = io.WriteString(w, `{}`)
			})
			if err := tc.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCardGetAndDelete(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/cards/c1" {
				t.Fatalf("request = %s %s", r.Method, r.URL.Path)
			}
			_, _ = io.WriteString(w, `{"id":"c1"}`)
		})
		if _, err := client.Card(context.Background(), "c1"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/cards/c1" {
				t.Fatalf("request = %s %s", r.Method, r.URL.Path)
			}
			_, _ = io.WriteString(w, `{}`)
		})
		if _, err := client.DeleteCard(context.Background(), "c1"); err != nil {
			t.Fatal(err)
		}
	})
}
