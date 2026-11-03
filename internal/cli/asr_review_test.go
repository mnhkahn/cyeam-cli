package cli

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/reading"
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
func TestASRReviewGeneratesReplayableReportOffline(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	ref, hyp := readingFiles(t)
	feedback := filepath.Join(t.TempDir(), "feedback.txt")
	var out bytes.Buffer
	cmd := NewRootCommand(Dependencies{Stdout: &out})
	cmd.SetArgs([]string{"asr", "review", "--reference", ref, "--transcript", hyp, "--confirmed-grade", "中", "--feedback-out", feedback})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	r, err := reading.ParseReport(out.Bytes())
	if err != nil {
		t.Fatal(err, out.String())
	}
	saved, err := os.ReadFile(feedback)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != r.Feedback || r.Grade != "中" || r.Provisional || len(r.FocusWords) != 2 {
		t.Fatal(r.Feedback)
	}
	if strings.Contains(r.Feedback, "你读错") {
		t.Fatal(r.Feedback)
	}
}
func TestASRReviewCanTranscribeAudioAndPrintFeedback(t *testing.T) {
	ref, _ := readingFiles(t)
	audio := filepath.Join(t.TempDir(), "reading.m4a")
	if err := os.WriteFile(audio, []byte("fake-audio"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ref, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}
	fake := &fakeTranscriber{}
	var out bytes.Buffer
	cmd := newASRReviewCommand(fake)
	cmd.Flags().Bool("pretty", true, "")
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--reference", ref, "--audio", audio})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if fake.filename != audio || !strings.Contains(out.String(), "暂评·优") {
		t.Fatal(out.String())
	}
}
func TestASRReviewRejectsConflictingOrDestructiveFlags(t *testing.T) {
	ref, hyp := readingFiles(t)
	for _, extra := range [][]string{{"--audio", "unused.m4a"}, {"--confirmed-grade", "A"}, {"--feedback-out", ref}, {"--feedback-out", hyp}} {
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
	r, err := reading.Review("Robots help people.", "Robots help people.", reading.Options{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(r)
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{{"--text", "invented"}, {"--grade", "中"}, {"--provisional"}} {
		cmd := newTrelloReviewCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(append([]string{"card", "--reading-report", path}, extra...))
		err := cmd.Execute()
		if err == nil || strings.Contains(err.Error(), "not logged in") {
			t.Fatal(extra, err)
		}
	}
}

func TestReadingReportPublishesFixedFeedbackAndGrade(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "test-key")
	t.Setenv("TRELLO_TOKEN", "test-token")
	r, err := reading.Review("Robotics club loves robots. Robotics club makes robots.", "Robot club loves robot. Robot club makes robot.", reading.Options{ConfirmedGrade: "中"})
	if err != nil {
		t.Fatal(err)
	}
	expected := r.Feedback
	// Generated fields are not authoritative when replaying a saved report.
	r.Feedback = "model-invented feedback"
	r.Grade = "100"
	data, _ := json.Marshal(r)
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
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
			if err := req.ParseForm(); err != nil {
				t.Error(err)
			}
			updatedDescription = req.Form.Get("desc")
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
		cmd := newTrelloReviewCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"card", "--reading-report", path, "--review-id", reviewID})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	run()
	run()
	if posts != 1 || updatedDescription != "打分结果：中\nOriginal assignment" {
		t.Fatal(posts, updatedDescription)
	}
	var payload struct {
		Grade       string `json:"grade"`
		Provisional bool   `json:"provisional"`
		Text        string `json:"text"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(publishedText, "作业批改：\n")), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Grade != "中" || payload.Provisional || payload.Text != expected {
		t.Fatal(publishedText)
	}
}
