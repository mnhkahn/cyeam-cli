package onedrive

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestListFolderRequestsAndParsesWebURL(t *testing.T) {
	var requestedSelect string
	client := &Client{
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedSelect = req.URL.Query().Get("$select")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"value": [{
						"id": "note-id",
						"name": "日记.html",
						"lastModifiedDateTime": "2026-06-12T10:00:00Z",
						"webUrl": "https://onedrive.live.com/view.aspx?resid=note-id"
					}]
				}`)),
				Header: make(http.Header),
			}, nil
		})},
		tokenFunc: func(ctx context.Context) (string, error) {
			return "token", nil
		},
	}

	items, err := client.ListFolder(context.Background(), "Notes")
	if err != nil {
		t.Fatalf("ListFolder: %v", err)
	}

	if !strings.Contains(requestedSelect, "webUrl") {
		t.Fatalf("$select = %q, want webUrl", requestedSelect)
	}
	if len(items) != 1 {
		t.Fatalf("items length = %d", len(items))
	}
	if items[0].WebURL != "https://onedrive.live.com/view.aspx?resid=note-id" {
		t.Fatalf("web url = %q", items[0].WebURL)
	}
}
