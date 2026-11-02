package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
)

var parallelPaddle = []byte(`{"ocrResults":[{"prunedResult":{"rec_texts":["(1)12*3=35"]}}]}`)
var parallelVision = []byte(`{"lines":[{"text":"(1)12*3=34","page":1,"box":[0,0,120,20]}]}`)

func TestArithmeticParallelStartsBothBeforeWaiting(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	jobs := []arithmeticRecognizer{}
	for _, engine := range []string{"paddleocr", "vision"} {
		engine := engine
		jobs = append(jobs, arithmeticRecognizer{engine, func(ctx context.Context) ([]byte, error) {
			started <- engine
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if engine == "vision" {
				return parallelVision, nil
			}
			return parallelPaddle, nil
		}})
	}
	type result struct {
		data []byte
		err  error
	}
	finished := make(chan result, 1)
	go func() {
		d, e := recognizeArithmeticParallel(context.Background(), time.Second, jobs)
		finished <- result{d, e}
	}()
	for range jobs {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("providers not running concurrently")
		}
	}
	close(release)
	r := <-finished
	if r.err != nil {
		t.Fatal(r.err)
	}
	report, err := checkArithmeticEvidence(r.data, arithmetic.CheckOptions{})
	if err != nil || report.Incorrect != 1 || report.Questions[0].SubmittedAnswer != "34" {
		t.Fatalf("Vision preference lost: %+v %v", report, err)
	}
}
func TestArithmeticParallelSurvivesEitherTimeoutOrFailure(t *testing.T) {
	for _, failed := range []string{"vision", "paddleocr"} {
		for _, timeout := range []bool{false, true} {
			jobs := []arithmeticRecognizer{}
			for _, engine := range []string{"vision", "paddleocr"} {
				engine := engine
				jobs = append(jobs, arithmeticRecognizer{engine, func(ctx context.Context) ([]byte, error) {
					if engine == failed {
						if timeout {
							<-ctx.Done()
							return nil, ctx.Err()
						}
						return nil, errors.New("unavailable")
					}
					if engine == "vision" {
						return parallelVision, nil
					}
					return parallelPaddle, nil
				}})
			}
			data, err := recognizeArithmeticParallel(context.Background(), 30*time.Millisecond, jobs)
			if err != nil {
				t.Fatal(err)
			}
			r, err := checkArithmeticEvidence(data, arithmetic.CheckOptions{})
			if err != nil || r.Detected != 1 || r.Engine == failed {
				t.Fatalf("%+v %v", r, err)
			}
			if len(r.Warnings) < 2 {
				t.Fatal("failure was not retained")
			}
		}
	}
}
func TestArithmeticParallelRejectsBothFailuresAndHonorsCancellation(t *testing.T) {
	bad := arithmeticRecognizer{"vision", func(context.Context) ([]byte, error) { return []byte(`{}`), nil }}
	if _, err := recognizeArithmeticParallel(context.Background(), time.Second, []arithmeticRecognizer{bad}); err == nil {
		t.Fatal("accepted empty OCR")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := recognizeArithmeticParallel(ctx, time.Second, []arithmeticRecognizer{bad}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestArithmeticEnsembleReplayRetainsHashAndBothReadings(t *testing.T) {
	dir := t.TempDir()
	image := filepath.Join(dir, "photo.jpg")
	src := []byte("hash-only replay input")
	if err := os.WriteFile(image, src, 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(arithmeticOCREvidence{Version: 1, Providers: []arithmeticOCRProvider{{Engine: "vision", Raw: parallelVision}, {Engine: "paddleocr", Raw: parallelPaddle}}})
	data, _, err := stampArithmeticInput(raw, src, src, image, 0)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ocr.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd := newArithmeticCheckCommandWithVision(false, nil)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{image, "--ocr-result", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var r struct {
		arithmetic.CheckReport
		Input arithmeticInput `json:"input"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if !r.Input.Verified || r.Input.Mode != "vision+paddleocr" || len(r.Questions[0].Readings) != 2 {
		t.Fatal(out.String())
	}
	if err := os.WriteFile(image, []byte("different"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = newArithmeticCheckCommandWithVision(false, nil)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{image, "--ocr-result", path})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "different image") {
		t.Fatal(err)
	}
}
func TestArithmeticVisionKeepsOriginalUnlessAlternativePasses(t *testing.T) {
	raw := []byte(`{"lines":[{"text":"(1)12*3=t","page":1}],"alternatives":[{"kind":"deskew","line":{"text":"(1)12*3=35","page":1}}]}`)
	r, err := checkArithmeticProvider(arithmeticOCRProvider{Engine: "vision", Raw: raw}, arithmetic.CheckOptions{})
	if err != nil || r.NeedsReview != 1 || r.Questions[0].SubmittedAnswer != "t" {
		t.Fatalf("%+v %v", r, err)
	}
	raw = bytes.Replace(raw, []byte("=35"), []byte("=36"), 1)
	r, err = checkArithmeticProvider(arithmeticOCRProvider{Engine: "vision", Raw: raw}, arithmetic.CheckOptions{})
	if err != nil || r.Correct != 1 || r.Detected != 1 || len(r.Questions[0].Readings) != 2 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestArithmeticPrettyDoesNotDiscloseExpectedAnswer(t *testing.T) {
	r, err := checkArithmeticEvidence(parallelPaddle, arithmetic.CheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeArithmeticCheck(&out, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "36") || strings.Contains(out.String(), "计算结果：") {
		t.Fatal(out.String())
	}
}

func TestArithmeticCheckUsesVisionWhenPaddleConfigurationFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.png")
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, imageData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	native := func(ctx context.Context, src []byte) ([]byte, error) {
		called = true
		if !bytes.Equal(src, imageData.Bytes()) {
			t.Error("input modified")
		}
		return parallelVision, nil
	}
	var out bytes.Buffer
	cmd := newArithmeticCheckCommandWithVision(true, native)
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{path, "--ocr-env", filepath.Join(dir, "missing.env")})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var report arithmetic.CheckReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !called || report.Engine != "vision" || report.Detected != 1 {
		t.Fatal(out.String())
	}
}
