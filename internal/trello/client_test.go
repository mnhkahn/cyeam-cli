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

func TestBoardStatusChangesRequestsListMoveActions(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/boards/board-1/actions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		for key, want := range map[string]string{
			"filter":               "updateCard:idList",
			"since":                "2026-07-28T16:00:00Z",
			"before":               "2026-07-29T16:00:00Z",
			"limit":                "1000",
			"memberCreator":        "true",
			"memberCreator_fields": "fullName,username",
		} {
			if q.Get(key) != want {
				t.Fatalf("%s = %q, want %q", key, q.Get(key), want)
			}
		}
		_, _ = io.WriteString(w, `[]`)
	})
	if _, err := client.BoardStatusChanges(context.Background(), "board-1", "2026-07-28T16:00:00Z", "2026-07-29T16:00:00Z", 1000); err != nil {
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

func TestDownloadAttachmentUsesAuthenticatedDownloadEndpoint(t *testing.T) {
	requests := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/cards/card-1/attachments/attachment-1":
			if r.URL.Query().Get("key") != "key" || r.URL.Query().Get("token") != "token" {
				t.Fatalf("missing metadata credentials: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"id":"attachment-1","name":"result photo.jpg","mimeType":"image/jpeg"}`)
		case "/cards/card-1/attachments/attachment-1/download/result photo.jpg":
			wantAuthorization := `OAuth oauth_consumer_key="key", oauth_token="token"`
			if got := r.Header.Get("Authorization"); got != wantAuthorization {
				t.Fatalf("Authorization = %q, want %q", got, wantAuthorization)
			}
			if r.URL.Query().Get("key") != "" || r.URL.Query().Get("token") != "" {
				t.Fatalf("download credentials leaked in query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("jpeg-bytes"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	download, err := client.DownloadAttachment(context.Background(), "card-1", "attachment-1", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || download.Name != "result photo.jpg" || download.MIMEType != "image/jpeg" || string(download.Data) != "jpeg-bytes" {
		t.Fatalf("download = %#v, requests = %d", download, requests)
	}
}

func TestDownloadAttachmentRejectsOversizedContent(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/card-1/attachments/attachment-1":
			_, _ = io.WriteString(w, `{"id":"attachment-1","name":"photo.jpg","mimeType":"image/jpeg"}`)
		default:
			_, _ = w.Write([]byte("too large"))
		}
	})
	if _, err := client.DownloadAttachment(context.Background(), "card-1", "attachment-1", 3); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %v, want size limit error", err)
	}
}

func TestProcessCardBuildsDownloadLinks(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/abc12345":
			_, _ = io.WriteString(w, `{"id":"card-1","name":"数学作业"}`)
		case "/cards/card-1/attachments":
			_, _ = io.WriteString(w, `[
  {"id":"attachment-1","name":"photo 1.jpg","mimeType":"image/jpeg","url":"https://trello.com/1/cards/card-1/attachments/attachment-1/download/photo%201.jpg"},
  {"id":"attachment-2","name":"photo2.png","mimeType":"image/png"}
]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	results, err := client.ProcessCard(context.Background(), "https://trello.com/c/abc12345/1-card")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	first := results[0]
	if first.FileName != "photo 1.jpg" || first.MimeType != "image/jpeg" {
		t.Fatalf("first = %#v", first)
	}
	if first.DownloadURL != "https://trello.com/1/cards/card-1/attachments/attachment-1/download/photo%201.jpg" {
		t.Fatalf("DownloadURL = %q", first.DownloadURL)
	}
	// url 字段缺失时按固定格式拼接
	second := results[1]
	if second.DownloadURL != "https://trello.com/1/cards/card-1/attachments/attachment-2/download/photo2.png" {
		t.Fatalf("DownloadURL = %q", second.DownloadURL)
	}
}

func TestExtractShortID(t *testing.T) {
	for _, rawURL := range []string{"https://trello.com/c/abc12345", "https://trello.com/c/abc12345/some-name"} {
		shortID, err := ExtractShortID(rawURL)
		if err != nil || shortID != "abc12345" {
			t.Fatalf("ExtractShortID(%q) = %q, %v", rawURL, shortID, err)
		}
	}
	for _, rawURL := range []string{"https://trello.com/b/boards", "https://trello.com/c/short", "not a url"} {
		if _, err := ExtractShortID(rawURL); err == nil {
			t.Fatalf("ExtractShortID(%q) should fail", rawURL)
		}
	}
}

func TestDownloadAttachmentSizedPicksPreviewWithinMaxWidth(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/card-1/attachments/attachment-1":
			_, _ = io.WriteString(w, `{"id":"attachment-1","name":"photo.jpg","mimeType":"image/jpeg","previews":[
  {"id":"prev-70","width":70,"height":50,"url":"https://trello.com/1/cards/card-1/attachments/attachment-1/previews/prev-70/download/photo.webp"},
  {"id":"prev-600","width":600,"height":450,"url":"https://trello.com/1/cards/card-1/attachments/attachment-1/previews/prev-600/download/photo.webp"},
  {"id":"prev-1200","width":1200,"height":900,"url":"https://trello.com/1/cards/card-1/attachments/attachment-1/previews/prev-1200/download/photo.webp"}
]}`)
		case "/cards/card-1/attachments/attachment-1/previews/prev-600/download/photo.webp":
			if got := r.Header.Get("Authorization"); got == "" {
				t.Fatal("preview download missing OAuth header")
			}
			w.Header().Set("Content-Type", "image/webp")
			_, _ = w.Write([]byte("webp-bytes"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	download, err := client.DownloadAttachmentSized(context.Background(), "card-1", "attachment-1", 1000, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if download.Name != "photo.webp" || download.MIMEType != "image/webp" || string(download.Data) != "webp-bytes" {
		t.Fatalf("download = %#v", download)
	}
}

func TestDownloadAttachmentSizedFallsBackToSmallestPreview(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cards/card-1/attachments/attachment-1":
			_, _ = io.WriteString(w, `{"id":"attachment-1","name":"photo.jpg","mimeType":"image/jpeg","previews":[
  {"id":"prev-600","width":600,"height":450,"url":"https://trello.com/1/cards/card-1/attachments/attachment-1/previews/prev-600/download/photo.webp"},
  {"id":"prev-1200","width":1200,"height":900,"url":"https://trello.com/1/cards/card-1/attachments/attachment-1/previews/prev-1200/download/photo.webp"}
]}`)
		case "/cards/card-1/attachments/attachment-1/previews/prev-600/download/photo.webp":
			w.Header().Set("Content-Type", "image/webp")
			_, _ = w.Write([]byte("webp-bytes"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	if _, err := client.DownloadAttachmentSized(context.Background(), "card-1", "attachment-1", 100, 1024); err != nil {
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
			call:      func(c *Client) error { _, err := c.ArchiveList(context.Background(), "l1"); return err },
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
