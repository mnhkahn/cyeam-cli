package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetJSONEscapesQueryAndReturnsBody(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	body, err := c.GetJSON(context.Background(), "/search/api", map[string]string{"q": "golang 优化"})
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", body)
	}
	if gotPath != "/search/api?q=golang+%E4%BC%98%E5%8C%96" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestPostRawSendsBodyAndReturnsBody(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		_, _ = w.Write([]byte(`{"id":"abc123"}`))
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	body, err := c.PostRaw(context.Background(), "/api/roadbook/share", nil, []byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("PostRaw: %v", err)
	}
	if gotBody != `{"items":[]}` {
		t.Fatalf("got body = %q", gotBody)
	}
	if string(body) != `{"id":"abc123"}` {
		t.Fatalf("response = %s", body)
	}
}

func TestStreamGETCopiesResponseToWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("chunk one\nchunk two\n"))
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	var out strings.Builder
	if err := c.StreamGET(context.Background(), "/ai/architecture", map[string]string{"q": "限流", "mode": "fast"}, &out); err != nil {
		t.Fatalf("StreamGET: %v", err)
	}
	if out.String() != "chunk one\nchunk two\n" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestDownloadBinaryReturnsBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	body, err := c.DownloadBinary(context.Background(), "/mo/api/char/compose", map[string]string{"char": "曦"})
	if err != nil {
		t.Fatalf("DownloadBinary: %v", err)
	}
	if string(body) != string([]byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatalf("body = %v", body)
	}
}

func TestUploadFileUsesMultipartFieldImage(t *testing.T) {
	var gotFilename string
	var gotValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("FormFile image: %v", err)
		}
		defer file.Close()
		gotFilename = header.Filename
		buf, _ := io.ReadAll(file)
		gotValue = string(buf)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	body, err := c.UploadFile(context.Background(), "/mo/api/ocr", "image", "sample.png", []byte("png-data"))
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if gotFilename != "sample.png" {
		t.Fatalf("filename = %q", gotFilename)
	}
	if gotValue != "png-data" {
		t.Fatalf("value = %q", gotValue)
	}
	if string(body) != `{"code":0}` {
		t.Fatalf("response = %s", body)
	}
}

func TestNon2xxReturnsStatusAndBodyExcerpt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request details", http.StatusBadRequest)
	}))
	defer server.Close()

	c := New(server.URL, server.Client())
	_, err := c.GetJSON(context.Background(), "/bad", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad request details") {
		t.Fatalf("error = %q", err.Error())
	}
}
