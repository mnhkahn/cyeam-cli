package cyeam

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/client"
)

func TestServiceMapsDateAndRoadbook(t *testing.T) {
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		switch r.URL.Path {
		case "/api/calendar/dates":
			_, _ = w.Write([]byte(`{"2026-06-09":{"date":"2026-06-09","type":0,"is_holiday":false,"lunar":"四月廿四"}}`))
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
		"/api/calendar/dates?year=2026",
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

func TestDateHolidayReturnsCalendarRecordUnchanged(t *testing.T) {
	// Include an unknown future field: the client must preserve the web record.
	record := `{"date":"2027-01-01","type":2,"name":"元旦","week":5,"is_holiday":true,"is_adjusted_workday":false,"lunar":"冬月廿四","term":"","festivals":["元旦节"],"future_field":{"value":1}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/api/calendar/dates?year=2027" {
			t.Errorf("path = %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"2027-01-01":`+record+`}`)
	}))
	defer server.Close()
	svc := NewService(client.New(server.URL, server.Client()), server.URL)
	got, err := svc.DateHoliday(context.Background(), "2027-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != record {
		t.Fatalf("record changed: %s", got)
	}
}

func TestDateHolidayRejectsUnavailableCalendarData(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `<html>404</html>`, `{"error":"unavailable"}`, `{"2027-01-01":null}`, `{"2027-01-01":{"date":"2027-01-01"}}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
			defer server.Close()
			svc := NewService(client.New(server.URL, server.Client()), server.URL)
			if _, err := svc.DateHoliday(context.Background(), "2027-01-01"); err == nil {
				t.Fatal("expected error instead of a guessed workday")
			}
		})
	}
}

func TestDateHolidayRejectsInvalidDateBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected request") }))
	defer server.Close()
	svc := NewService(client.New(server.URL, server.Client()), server.URL)
	for _, date := range []string{"bad", "2027-02-30", "1899-12-31", "2101-01-01"} {
		if _, err := svc.DateHoliday(context.Background(), date); err == nil {
			t.Errorf("date %s should fail", date)
		}
	}
}

func TestDateHolidayPropagatesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "not deployed", http.StatusNotFound) }))
	defer server.Close()
	svc := NewService(client.New(server.URL, server.Client()), server.URL)
	if _, err := svc.DateHoliday(context.Background(), "2027-01-01"); err == nil {
		t.Fatal("expected HTTP error")
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
	if _, err := svc.MoGuwen(context.Background(), "兰亭序", true, "行书"); err != nil {
		t.Fatalf("MoGuwen: %v", err)
	}
	if _, err := svc.MoCharDetail(context.Background(), "之", "行书"); err != nil {
		t.Fatalf("MoCharDetail: %v", err)
	}
	if _, err := svc.MoCharComposition(context.Background(), "曦", "行书"); err != nil {
		t.Fatalf("MoCharComposition: %v", err)
	}
	if _, err := svc.MoCharCompose(context.Background(), "曦", "行书"); err != nil {
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
