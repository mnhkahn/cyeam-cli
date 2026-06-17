package cyeam

import (
	"context"
	"encoding/json"
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
		case "/api/holiday/year/2026":
			_, _ = w.Write([]byte(`{"code":0,"holiday":{},"type":{}}`))
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
	svc.holidayClient = client.New(server.URL, server.Client())

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
		"/api/holiday/year/2026?type=Y&weekday=Y",
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

func TestDateHolidayUsesTimorYearEndpoint(t *testing.T) {
	var gotPath string
	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		gotUserAgent = r.UserAgent()
		_, _ = w.Write([]byte(`{
			"code":0,
			"holiday":{
				"06-19":{"holiday":true,"name":"端午节","wage":3,"date":"2026-06-19","rest":8}
			},
			"type":{
				"2026-06-19":{"type":2,"name":"端午节","week":5}
			}
		}`))
	}))
	defer server.Close()

	svc := NewService(client.New("https://www.cyeam.com", server.Client()), "https://www.cyeam.com")
	svc.holidayClient = client.New(server.URL, server.Client())
	body, err := svc.DateHoliday(context.Background(), "2026-06-19")
	if err != nil {
		t.Fatalf("DateHoliday: %v", err)
	}

	var got struct {
		Date      string `json:"date"`
		IsHoliday bool   `json:"is_holiday"`
		Name      string `json:"name"`
		Type      int    `json:"type"`
		Week      int    `json:"week"`
		Wage      int    `json:"wage"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gotPath != "/api/holiday/year/2026?type=Y&weekday=Y" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotUserAgent == "" || gotUserAgent == "Go-http-client/1.1" {
		t.Fatalf("user agent = %q", gotUserAgent)
	}
	if got.Date != "2026-06-19" || !got.IsHoliday || got.Name != "端午节" || got.Type != 2 || got.Week != 5 || got.Wage != 3 {
		t.Fatalf("holiday = %+v", got)
	}
}

func TestDateHolidayHandlesAdjustedWorkday(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"code":0,
			"holiday":{
				"05-09":{"holiday":false,"name":"劳动节后补班","wage":1,"after":true,"target":"劳动节","date":"2026-05-09","rest":3}
			},
			"type":{
				"2026-05-09":{"type":3,"name":"劳动节后补班","week":6}
			}
		}`))
	}))
	defer server.Close()

	svc := NewService(client.New("https://www.cyeam.com", server.Client()), "https://www.cyeam.com")
	svc.holidayClient = client.New(server.URL, server.Client())
	body, err := svc.DateHoliday(context.Background(), "2026-05-09")
	if err != nil {
		t.Fatalf("DateHoliday: %v", err)
	}

	var got struct {
		Date      string `json:"date"`
		IsHoliday bool   `json:"is_holiday"`
		Name      string `json:"name"`
		Type      int    `json:"type"`
		Week      int    `json:"week"`
		After     *bool  `json:"after"`
		Target    string `json:"target"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Date != "2026-05-09" || got.IsHoliday || got.Name != "劳动节后补班" || got.Type != 3 || got.Week != 6 || got.After == nil || !*got.After || got.Target != "劳动节" {
		t.Fatalf("holiday = %+v", got)
	}
}

func TestDateHolidayHandlesRegularWeekdayAndWeekend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"holiday":{},"type":{}}`))
	}))
	defer server.Close()

	svc := NewService(client.New("https://www.cyeam.com", server.Client()), "https://www.cyeam.com")
	svc.holidayClient = client.New(server.URL, server.Client())

	weekdayBody, err := svc.DateHoliday(context.Background(), "2026-06-09")
	if err != nil {
		t.Fatalf("DateHoliday weekday: %v", err)
	}
	var weekday struct {
		Date      string `json:"date"`
		IsHoliday bool   `json:"is_holiday"`
		Type      int    `json:"type"`
		Week      int    `json:"week"`
	}
	if err := json.Unmarshal(weekdayBody, &weekday); err != nil {
		t.Fatalf("unmarshal weekday: %v", err)
	}
	if weekday.Date != "2026-06-09" || weekday.IsHoliday || weekday.Type != 0 || weekday.Week != 2 {
		t.Fatalf("weekday = %+v", weekday)
	}

	weekendBody, err := svc.DateHoliday(context.Background(), "2026-06-14")
	if err != nil {
		t.Fatalf("DateHoliday weekend: %v", err)
	}
	var weekend struct {
		Date      string `json:"date"`
		IsHoliday bool   `json:"is_holiday"`
		Name      string `json:"name"`
		Type      int    `json:"type"`
		Week      int    `json:"week"`
	}
	if err := json.Unmarshal(weekendBody, &weekend); err != nil {
		t.Fatalf("unmarshal weekend: %v", err)
	}
	if weekend.Date != "2026-06-14" || !weekend.IsHoliday || weekend.Name != "周日" || weekend.Type != 1 || weekend.Week != 7 {
		t.Fatalf("weekend = %+v", weekend)
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
