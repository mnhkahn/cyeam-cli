package paddleocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ocrResponse = `{"errorCode":0,"result":{"ocrResults":[{"prunedResult":{"rec_texts":["2+2=4"],"rec_scores":[0.98],"rec_polys":[[[1,2],[40,2],[40,20],[1,20]]]}}]}}`

func TestClientProtocolAndConfig(t *testing.T) {
	image := []byte("test image bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "token test-secret" {
			t.Error("wrong protocol")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["file"] != base64.StdEncoding.EncodeToString(image) || body["fileType"] != float64(1) || body["useLayoutDetection"] != true || body["useOcrForImageBlock"] != true || body["useDocUnwarping"] != false || body["useDocOrientationClassify"] != true {
			t.Error("wrong payload")
		}
		for _, key := range []string{"promptLabel", "maxPixels", "maxNewTokens"} {
			if _, exists := body[key]; exists {
				t.Errorf("unexpected recognition override: %s", key)
			}
		}
		fmt.Fprint(w, ocrResponse)
	}))
	defer server.Close()
	t.Setenv("OCR_API_URL", server.URL)
	t.Setenv("OCR_API_TOKEN", "test-secret")
	client, err := Config("")
	if err != nil {
		t.Fatal(err)
	}
	lines, err := client.Recognize(context.Background(), image)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Text != "2+2=4" || *lines[0].Confidence != .98 || lines[0].Box != (Box{1, 2, 40, 20}) {
		t.Fatal(lines)
	}
}
func TestConfigDotenvAndPrecedence(t *testing.T) {
	mockCredentials(t)
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	file := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(file, []byte("# comment\nexport OCR_API_URL='https://example.invalid/ocr'\nOCR_API_TOKEN=\"file-secret\"\nIGNORED=$(touch /tmp/never-execute-dotenv)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Config(file)
	if err != nil || c.Token != "file-secret" {
		t.Fatal(err)
	}
	t.Setenv("OCR_API_TOKEN", "override")
	c, err = Config(file)
	if err != nil || c.Token != "override" {
		t.Fatal(err)
	}
	t.Setenv("OCR_API_URL", "https://user:secret@example.invalid")
	if _, err := Config(file); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("unsafe URL or error")
	}
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	if _, err := Config(""); err == nil {
		t.Fatal("missing config accepted")
	}
}
func TestClientFailuresDoNotExposeSecrets(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{401, "secret-token private-endpoint"}, {200, `{"errorCode":123,"errorMsg":"secret-token private-endpoint"}`}, {200, `not JSON`}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
		c := Client{URL: server.URL, Token: "secret-token"}
		_, err := c.Recognize(context.Background(), []byte("image"))
		server.Close()
		if err == nil || strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "private-endpoint") || strings.Contains(err.Error(), server.URL) {
			t.Fatal(err)
		}
	}
	// Refuse redirects rather than forwarding configured credentials.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.invalid", 302) }))
	defer server.Close()
	_, err := (&Client{URL: server.URL, Token: "secret"}).Recognize(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = (&Client{URL: server.URL, Token: "secret"}).Recognize(ctx, nil)
	if err != context.Canceled {
		t.Fatal(err)
	}
}
func TestParseLayoutBoxesMarkdownAndMalformed(t *testing.T) {
	for _, body := range []string{
		`{"layoutParsingResults":[{"prunedResult":{"spotting_res":{"rec_texts":["2+2=4"],"rec_polys":[[[0,0],[10,0],[10,10],[0,10]]]}}}]}`,
		`{"ocrResults":[{"prunedResult":{"rec_texts":["2+2=4"],"rec_boxes":[[0,0,10,10]]}}]}`,
		`{"layoutParsingResults":[{"prunedResult":{"parsing_res_list":[{"block_content":"2+2=4","block_bbox":[0,0,10,10]}]}}]}`,
		`{"layoutParsingResults":[{"markdown":{"text":"2+2=4"}}]}`,
	} {
		lines, err := Parse([]byte(body))
		if err != nil || len(lines) != 1 || lines[0].Text != "2+2=4" || lines[0].Confidence != nil {
			t.Fatal(lines, err)
		}
	}
	for _, body := range []string{`{}`, `not JSON`, `{"ocrResults":[]}`, `{"ocrResults":[{"prunedResult":{"rec_texts":["a","b"],"rec_polys":[[[0,0],[1,0],[1,1],[0,1]]]}}]}`} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatal(body)
		}
	}
}

func TestClientNeverUploadsWithoutToken(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()
	_, err := (&Client{URL: server.URL}).Recognize(context.Background(), []byte("image"))
	if !errors.Is(err, ErrNotConfigured) || called {
		t.Fatal("missing token allowed an OCR request", err)
	}
}

func TestParseRejectsSuccessfulButEmptySpottingResult(t *testing.T) {
	data := []byte(`{"errorCode":0,"result":{"layoutParsingResults":[{"prunedResult":{"spotting_res":{"rec_texts":[],"rec_polys":[]},"parsing_res_list":[{"block_label":"spotting","block_content":""}]}}]}}`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "no text") {
		t.Fatal("empty response treated as successful OCR", err)
	}
}
func TestParseEmptyBlocksFallBackToMarkdownAndSplitText(t *testing.T) {
	data := []byte(`{"layoutParsingResults":[{"prunedResult":{"parsing_res_list":[{"block_content":""}]},"markdown":{"text":"(1)2+2=4\n(2)3+3=6"}}]}`)
	lines, err := Parse(data)
	if err != nil || len(lines) != 2 {
		t.Fatal(lines, err)
	}
	data = []byte(`{"layoutParsingResults":[{"prunedResult":{"parsing_res_list":[{"block_content":"(1)2+2=4\n\n(2)3+3=6","block_bbox":[0,0,100,100]}]}}]}`)
	lines, err = Parse(data)
	if err != nil || len(lines) != 2 || lines[1].Text != "(2)3+3=6" {
		t.Fatal(lines, err)
	}
}

func TestOCRCanSkipOrientationAndMeasureRequestPhases(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["useDocUnwarping"] != true {
			t.Error("unwarping was not enabled")
		}
		if body["useDocOrientationClassify"] != false {
			t.Error("orientation was not disabled")
		}
		fmt.Fprint(w, ocrResponse)
	}))
	defer server.Close()
	var timing RequestTiming
	client := Client{URL: server.URL, Token: "test", SkipOrientation: true, Unwarp: true, Timing: &timing}
	if _, err := client.Recognize(context.Background(), []byte("image")); err != nil {
		t.Fatal(err)
	}
	if !called || timing.TotalMS <= 0 || timing.PrepareConnectMS < 0 || timing.UploadMS < 0 || timing.WaitMS < 0 || timing.DownloadMS < 0 {
		t.Fatal(timing)
	}
}
