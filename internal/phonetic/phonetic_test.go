package phonetic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(server *httptest.Server) *Client {
	return &Client{
		BaseURL:      server.URL + "/search",
		HTTPClient:   server.Client(),
		MaxBodyBytes: 1024,
	}
}

func TestClientFetchEncodesQueryAndParsesResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "C++" {
			t.Fatalf("query = %q, want C++", got)
		}
		_, _ = w.Write([]byte(`<div class="baav"><span class="pronounce"><span class="phonetic">/siː/</span></span><span class="pronounce"><span class="phonetic">/si/</span></span></div><div id="phrsListTab"><div class="trans-container"><ul><li>n. C plus plus</li></ul></div></div>`))
	}))
	defer server.Close()

	result, err := testClient(server).Fetch(context.Background(), " C++ ")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.Word != "C++" || result.UKPhonetic != "/siː/" || result.USPhonetic != "/si/" {
		t.Fatalf("result = %#v", result)
	}
	if got := strings.Join(result.Definitions, ","); got != "n. C plus plus" {
		t.Fatalf("definitions = %q", got)
	}
}

func TestClientFetchRejectsEmptyAndUnparsedResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body>not found</body></html>"))
	}))
	defer server.Close()

	client := testClient(server)
	if _, err := client.Fetch(context.Background(), "   "); err == nil {
		t.Fatal("Fetch() empty word succeeded")
	}
	if _, err := client.Fetch(context.Background(), "missing"); err == nil {
		t.Fatal("Fetch() unparsed result succeeded")
	}
}

func TestClientFetchLimitsResponseAndTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "slow" {
			time.Sleep(50 * time.Millisecond)
			return
		}
		_, _ = w.Write([]byte(strings.Repeat("x", 64)))
	}))
	defer server.Close()

	client := testClient(server)
	client.MaxBodyBytes = 32
	if _, err := client.Fetch(context.Background(), "large"); err == nil {
		t.Fatal("Fetch() oversized response succeeded")
	}

	client.MaxBodyBytes = 1024
	client.HTTPClient = &http.Client{Timeout: 10 * time.Millisecond}
	if _, err := client.Fetch(context.Background(), "slow"); err == nil {
		t.Fatal("Fetch() slow response succeeded")
	}
}
