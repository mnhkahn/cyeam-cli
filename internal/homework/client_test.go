package homework

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientHomeworkEndpoints(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/homework/review":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			if r.FormValue("kind") != "dictation" || r.FormValue("count") != "2" || r.FormValue("words") != "breakfast\nbiscuit" {
				t.Errorf("unexpected fields: %+v", r.MultipartForm.Value)
			}
			file, header, err := r.FormFile("image")
			if err != nil {
				t.Error(err)
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			if header.Filename != "words.png" || string(data) != "image" {
				t.Errorf("unexpected upload: %s %q", header.Filename, data)
			}
		case "/api/homework/regrade", "/api/homework/reading/regrade":
			data, _ := io.ReadAll(r.Body)
			if !json.Valid(data) {
				t.Errorf("invalid JSON: %q", data)
			}
		case "/api/homework/reading":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			if r.FormValue("reference") != "Robots build cars." {
				t.Errorf("reference = %q", r.FormValue("reference"))
			}
			file, header, err := r.FormFile("audio")
			if err != nil {
				t.Error(err)
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			if header.Filename != "reading.m4a" || string(data) != "audio" {
				t.Errorf("unexpected upload: %s %q", header.Filename, data)
			}
		default:
			http.NotFound(w, r)
		}
		_, _ = io.WriteString(w, `{"input":{"kind":"reading"},"report":{"feedback":"服务端反馈","future_field":42}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, server.Client())
	ctx := context.Background()
	if _, err := client.Review(ctx, ReviewRequest{Kind: "dictation", Count: 2, Words: []string{"breakfast", "biscuit"}, Filename: "words.png", Image: []byte("image")}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Regrade(ctx, json.RawMessage(`{"kind":"dictation"}`)); err != nil {
		t.Fatal(err)
	}
	result, err := client.Reading(ctx, "Robots build cars.", "reading.m4a", []byte("audio"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), `"future_field":42`) {
		t.Fatal("server fields were not preserved")
	}
	if _, err := client.ReadingRegrade(ctx, json.RawMessage(`{"reference":"Robots build cars.","transcript":"Robots build cars."}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"/api/homework/review", "/api/homework/regrade", "/api/homework/reading", "/api/homework/reading/regrade"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %v", calls)
		}
	}
}

func TestClientReturnsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":"请稍后再试"}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, server.Client())
	_, err := client.Regrade(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "请稍后再试") {
		t.Fatal(err)
	}
}
