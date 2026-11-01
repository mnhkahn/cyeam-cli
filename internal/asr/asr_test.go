package asr

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeUploadsAudioAndParsesText(t *testing.T) {
	var gotField, gotFile, gotContentType string
	var gotBytes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		file, header, err := r.FormFile("audio")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		gotField = "audio"
		gotFile = header.Filename
		gotContentType = header.Header.Get("Content-Type")
		gotBytes = len(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"Robotics Club is cool!"}`))
	}))
	defer server.Close()

	c := NewClient()
	c.Endpoint = server.URL
	result, err := c.Transcribe(context.Background(), "/tmp/reading.m4a", []byte("fake-audio"))
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if result.Text != "Robotics Club is cool!" {
		t.Fatalf("text = %q", result.Text)
	}
	if gotField != "audio" || gotFile != "reading.m4a" || gotBytes != len("fake-audio") {
		t.Fatalf("upload field=%q file=%q bytes=%d", gotField, gotFile, gotBytes)
	}
	if gotContentType != "audio/mp4" {
		t.Fatalf("content-type = %q", gotContentType)
	}
}

func TestTranscribeReturnsServerErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"语音识别服务暂不可用，请稍后再试"}`))
	}))
	defer server.Close()

	c := NewClient()
	c.Endpoint = server.URL
	_, err := c.Transcribe(context.Background(), "a.m4a", []byte("fake-audio"))
	if err == nil || !strings.Contains(err.Error(), "语音识别服务暂不可用") {
		t.Fatalf("err = %v", err)
	}
}

func TestTranscribeRejectsOversizedAudio(t *testing.T) {
	c := NewClient()
	c.MaxFileBytes = 4
	if _, err := c.Transcribe(context.Background(), "a.m4a", []byte("12345")); err == nil {
		t.Fatal("expected size error")
	}
}

func TestTranscribeRejectsEmptyAudio(t *testing.T) {
	c := NewClient()
	if _, err := c.Transcribe(context.Background(), "a.m4a", nil); err == nil {
		t.Fatal("expected empty file error")
	}
}
