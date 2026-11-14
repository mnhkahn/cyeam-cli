package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func readingFiles(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	ref := filepath.Join(dir, "reference.txt")
	hyp := filepath.Join(dir, "transcript.txt")
	for p, s := range map[string]string{ref: "Robotics club loves robots. Robotics club makes robots.", hyp: "Robot club loves robot. Robot club makes robot."} {
		if err := os.WriteFile(p, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return ref, hyp
}

func readingServerResponse(reference, transcript, feedback, grade string) json.RawMessage {
	data, _ := json.Marshal(map[string]any{
		"input":  map[string]any{"kind": "reading", "reference": reference, "transcript": transcript},
		"report": map[string]any{"schema_version": 1, "reference": reference, "transcript": transcript, "feedback": feedback, "grade": grade, "provisional": false, "focus": []any{}, "future_field": "preserved"},
	})
	return data
}

func TestASRReviewUsesServerForTranscript(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	ref, hyp := readingFiles(t)
	feedback := filepath.Join(t.TempDir(), "feedback.txt")
	service := &fakeHomeworkService{readingRegrade: func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
		var value struct {
			Kind       string `json:"kind"`
			Reference  string `json:"reference"`
			Transcript string `json:"transcript"`
		}
		if err := json.Unmarshal(input, &value); err != nil || value.Kind != "reading" || !strings.Contains(value.Reference, "Robotics") || !strings.Contains(value.Transcript, "Robot club") {
			t.Fatalf("input = %s", input)
		}
		return readingServerResponse(value.Reference, value.Transcript, "阅读等级：优。\n服务端固定反馈\n", "优"), nil
	}}
	var out bytes.Buffer
	cmd := NewRootCommand(Dependencies{Stdout: &out, Homework: service})
	cmd.SetArgs([]string{"asr", "review", "--reference", ref, "--transcript", hyp, "--feedback-out", feedback})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"future_field":"preserved"`) {
		t.Fatal(out.String())
	}
	saved, err := os.ReadFile(feedback)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "阅读等级：优。\n服务端固定反馈\n" {
		t.Fatal(string(saved))
	}
}

func TestASRReviewCanTranscribeAudioAndPrintServerFeedback(t *testing.T) {
	ref, _ := readingFiles(t)
	audio := filepath.Join(t.TempDir(), "reading.m4a")
	if err := os.WriteFile(audio, []byte("fake-audio"), 0600); err != nil {
		t.Fatal(err)
	}
	service := &fakeHomeworkService{reading: func(_ context.Context, reference, filename string, data []byte) (json.RawMessage, error) {
		if filename != audio || string(data) != "fake-audio" || !strings.Contains(reference, "Robotics") {
			t.Fatalf("unexpected reading request")
		}
		return readingServerResponse(reference, "server transcript", "阅读等级：良。\n由服务端生成\n", "良"), nil
	}}
	var out bytes.Buffer
	cmd := newASRReviewCommand(service)
	cmd.Flags().Bool("pretty", true, "")
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--reference", ref, "--audio", audio})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "阅读等级：良。\n由服务端生成\n" {
		t.Fatal(out.String())
	}
}

func TestASRReviewRejectsConflictingOrDestructiveFlags(t *testing.T) {
	ref, hyp := readingFiles(t)
	for _, extra := range [][]string{{"--audio", "unused.m4a"}, {"--confirmed-grade", "中"}, {"--feedback-out", ref}, {"--feedback-out", hyp}} {
		cmd := newASRReviewCommand(nil)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(append([]string{"--reference", ref, "--transcript", hyp}, extra...))
		if err := cmd.Execute(); err == nil {
			t.Fatal(extra)
		}
	}
	cmd := newASRReviewCommand(nil)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--reference", ref})
	if err := cmd.Execute(); err == nil {
		t.Fatal("accepted no transcript or audio")
	}
}

func TestReadingReportCannotCombineFreeformFeedbackOrGrade(t *testing.T) {
	data := readingServerResponse("Robots help people.", "Robots help people.", "阅读等级：优。\n", "优")
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{{"--text", "invented"}, {"--grade", "中"}, {"--provisional"}} {
		cmd := newTrelloReviewCommand(nil)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(append([]string{"card", "--reading-report", path}, extra...))
		err := cmd.Execute()
		if err == nil || strings.Contains(err.Error(), "not logged in") {
			t.Fatal(extra, err)
		}
	}
}

func TestReadingReportRegradesOnServerAndPublishesResult(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "test-key")
	t.Setenv("TRELLO_TOKEN", "test-token")
	reference := "Robotics club loves robots. Robotics club makes robots."
	transcript := "Robot club loves robot. Robot club makes robot."
	expected := "阅读等级：优。\n服务端权威反馈\n"
	data := readingServerResponse(reference, transcript, expected, "优")
	var tampered map[string]any
	if err := json.Unmarshal(data, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["report"].(map[string]any)["feedback"] = "model-invented feedback"
	tampered["report"].(map[string]any)["grade"] = "100"
	data, _ = json.Marshal(tampered)
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	regrades := 0
	service := &fakeHomeworkService{readingRegrade: func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
		regrades++
		var value struct {
			Reference  string `json:"reference"`
			Transcript string `json:"transcript"`
		}
		if err := json.Unmarshal(input, &value); err != nil || value.Reference != reference || value.Transcript != transcript {
			t.Fatalf("input = %s", input)
		}
		return readingServerResponse(reference, transcript, expected, "优"), nil
	}}
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	var publishedText, updatedDescription string
	posts := 0
	http.DefaultTransport = homeworkASRTransport(func(req *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == "GET" && req.URL.Path == "/1/cards/card/actions":
			recorder.WriteString(`[]`)
		case req.Method == "POST" && req.URL.Path == "/1/cards/card/actions/comments":
			if err := req.ParseForm(); err != nil {
				t.Error(err)
			}
			publishedText = req.Form.Get("text")
			posts++
			recorder.WriteString(`{"id":"comment"}`)
		case req.Method == "GET" && req.URL.Path == "/1/cards/card":
			recorder.WriteString(`{"desc":"Original assignment"}`)
		case req.Method == "PUT" && req.URL.Path == "/1/cards/card":
			updatedDescription = req.URL.Query().Get("desc")
			recorder.WriteString(`{"id":"card"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			recorder.WriteHeader(500)
		}
		return recorder.Result(), nil
	})
	reviewID := uuid.NewString()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(home, ".cyeam", "trello-reviews", reviewID+".json")
	t.Cleanup(func() { os.Remove(statePath); os.Remove(statePath + ".lock") })
	run := func() {
		cmd := newTrelloReviewCommand(service)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"card", "--reading-report", path, "--review-id", reviewID})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	run()
	run()
	if posts != 1 || regrades != 2 || updatedDescription != "打分结果：优\nOriginal assignment" {
		t.Fatal(posts, regrades, updatedDescription)
	}
	var payload struct {
		Grade       string `json:"grade"`
		Provisional bool   `json:"provisional"`
		Text        string `json:"text"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(publishedText, "作业批改：\n")), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Grade != "优" || payload.Provisional || payload.Text != expected {
		t.Fatal(publishedText)
	}
}
