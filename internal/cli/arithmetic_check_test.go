package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
)

func TestArithmeticCheckReplayThroughRoot(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	for _, pretty := range []bool{false, true} {
		var out bytes.Buffer
		cmd := NewRootCommand(Dependencies{Stdout: &out})
		args := []string{"arithmetic", "check", "unneeded-for-replay.png", "--ocr-result", "../arithmetic/testdata/paddle_spotting.json", "--expected-count", "18"}
		if pretty {
			args = append(args, "--pretty")
		}
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if pretty {
			if !strings.Contains(out.String(), "识别 18 题") || !strings.Contains(out.String(), "相符 10，不符 6，无法比较 2") || !strings.Contains(out.String(), "正确 10，错误 6，待核对 2") {
				t.Fatal(out.String())
			}
		} else {
			var envelope struct {
				OK   bool                   `json:"ok"`
				Data arithmetic.CheckReport `json:"data"`
			}
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if !envelope.OK || envelope.Data.Detected != 18 || envelope.Data.NeedsReview != 2 || envelope.Data.Correct != 10 || envelope.Data.Incorrect != 6 {
				t.Fatal(envelope)
			}
		}
	}
}
func TestArithmeticCheckRejectsInvalidFlagsBeforeUpload(t *testing.T) {
	for _, extra := range [][]string{{"--expected-count", "0"}, {"--expected-count", "-1"}, {"--min-confidence", "NaN"}, {"--min-confidence", "1.1"}, {"--retries", "-1"}, {"--retries", "6"}, {"--timeout", "0s"}, {"--timeout", "6m"}, {"--save-ocr", "image.png"}, {"--ocr-env", "config.env", "--save-ocr", "config.env"}, {"--ocr-result", "ocr.json", "--save-ocr", "ocr.json"}} {
		var out bytes.Buffer
		cmd := newArithmeticCheckCommand()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(append([]string{"image.png"}, extra...))
		if err := cmd.Execute(); err == nil {
			t.Fatal(extra)
		}
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.jpg")
	if err := os.WriteFile(bad, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newArithmeticCheckCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{bad})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "readable PNG") {
		t.Fatal(err)
	}
}

func TestArithmeticCheckUploadsOriginalPixels(t *testing.T) {
	var src bytes.Buffer
	if err := png.Encode(&src, image.NewRGBA(image.Rect(0, 0, 2200, 1200))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "original.png")
	if err := os.WriteFile(path, src.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["file"] != base64.StdEncoding.EncodeToString(src.Bytes()) {
			t.Error("original image was altered")
		}
		w.Write([]byte(`{"errorCode":0,"result":{"ocrResults":[{"prunedResult":{"rec_texts":["2+2=4"]}}]}}`))
	}))
	defer server.Close()
	t.Setenv("OCR_API_URL", server.URL)
	t.Setenv("OCR_API_TOKEN", "test")
	var out bytes.Buffer
	cmd := newArithmeticCheckCommandWithVision(false, nil)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{path, "--retries", "0"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("OCR was not called")
	}
	var result struct {
		Input arithmeticInput `json:"input"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Input.UploadWidth != 2200 || result.Input.UploadHeight != 1200 || result.Input.Mode != "layout" {
		t.Fatal(result.Input)
	}
}
