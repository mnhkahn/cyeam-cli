package cyeam

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/client"
)

func TestServiceMapsAskSearchDateAndRoadbook(t *testing.T) {
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		switch r.URL.Path {
		case "/search/api":
			_, _ = w.Write([]byte(`{"docs":[]}`))
		case "/api/date/slogan":
			_, _ = w.Write([]byte(`{"slogan":"ok"}`))
		case "/api/date/holiday":
			_, _ = w.Write([]byte(`{"is_holiday":false}`))
		case "/api/roadbook/share":
			body, _ := io.ReadAll(r.Body)
			if string(body) != `[{"name":"A"}]` {
				t.Fatalf("roadbook body = %q", body)
			}
			_, _ = w.Write([]byte(`{"id":"abc123"}`))
		case "/api/roadbook/get":
			_, _ = w.Write([]byte(`{"data":"[]"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(client.New(server.URL, server.Client()), server.URL)

	if _, err := svc.Search(context.Background(), "golang 优化"); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if _, err := svc.DateSlogan(context.Background(), "2026-06-09"); err != nil {
		t.Fatalf("DateSlogan: %v", err)
	}
	if _, err := svc.DateHoliday(context.Background(), "2026-06-09"); err != nil {
		t.Fatalf("DateHoliday: %v", err)
	}
	shared, err := svc.RoadbookShare(context.Background(), []byte(`[{"name":"A"}]`))
	if err != nil {
		t.Fatalf("RoadbookShare: %v", err)
	}
	if string(shared) != `{"id":"abc123","url":"`+server.URL+`/tool/roadbook?id=abc123"}` {
		t.Fatalf("shared = %s", shared)
	}
	if _, err := svc.RoadbookGet(context.Background(), "abc123"); err != nil {
		t.Fatalf("RoadbookGet: %v", err)
	}

	want := []string{
		"/search/api?q=golang+%E4%BC%98%E5%8C%96",
		"/api/date/slogan?date=2026-06-09",
		"/api/date/holiday?date=2026-06-09",
		"/api/roadbook/share",
		"/api/roadbook/get?id=abc123",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path %d = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestAskArchitectureStreamsFromArchitectureEndpoint(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_, _ = w.Write([]byte("answer"))
	}))
	defer server.Close()

	svc := NewService(client.New(server.URL, server.Client()), server.URL)
	var out testWriter
	if err := svc.AskArchitecture(context.Background(), "限流", "expert", &out); err != nil {
		t.Fatalf("AskArchitecture: %v", err)
	}
	if gotPath != "/ai/architecture?mode=expert&q=%E9%99%90%E6%B5%81" {
		t.Fatalf("path = %q", gotPath)
	}
	if out.String() != "answer" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestServiceMapsMoEndpointsWithXingshuFont(t *testing.T) {
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		switch r.URL.Path {
		case "/mo/api/guwen":
			_, _ = w.Write([]byte(`{"text":"兰亭序"}`))
		case "/mo/api/char/detail":
			_, _ = w.Write([]byte(`{"char":"之"}`))
		case "/mo/api/char/composition":
			_, _ = w.Write([]byte(`{"char":"曦"}`))
		case "/mo/api/char/compose":
			_, _ = w.Write([]byte{1, 2, 3})
		case "/mo/api/ocr":
			file, header, err := r.FormFile("image")
			if err != nil {
				t.Fatalf("FormFile image: %v", err)
			}
			defer file.Close()
			if header.Filename != "image.png" {
				t.Fatalf("filename = %q", header.Filename)
			}
			_, _ = w.Write([]byte(`{"code":0}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(client.New(server.URL, server.Client()), server.URL)
	if _, err := svc.MoGuwen(context.Background(), "兰亭序", true); err != nil {
		t.Fatalf("MoGuwen: %v", err)
	}
	if _, err := svc.MoCharDetail(context.Background(), "之"); err != nil {
		t.Fatalf("MoCharDetail: %v", err)
	}
	if _, err := svc.MoCharComposition(context.Background(), "曦"); err != nil {
		t.Fatalf("MoCharComposition: %v", err)
	}
	if _, err := svc.MoCharCompose(context.Background(), "曦"); err != nil {
		t.Fatalf("MoCharCompose: %v", err)
	}
	if _, err := svc.MoOCR(context.Background(), "image.png", []byte("png-data")); err != nil {
		t.Fatalf("MoOCR: %v", err)
	}

	want := []string{
		"/mo/api/guwen?compose=1&font=%E8%A1%8C%E4%B9%A6&text=%E5%85%B0%E4%BA%AD%E5%BA%8F",
		"/mo/api/char/detail?char=%E4%B9%8B&font=%E8%A1%8C%E4%B9%A6",
		"/mo/api/char/composition?char=%E6%9B%A6&font=%E8%A1%8C%E4%B9%A6",
		"/mo/api/char/compose?char=%E6%9B%A6&font=%E8%A1%8C%E4%B9%A6",
		"/mo/api/ocr",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path %d = %q, want %q", i, paths[i], want[i])
		}
	}
}

type testWriter struct {
	data []byte
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *testWriter) String() string {
	return string(w.data)
}
