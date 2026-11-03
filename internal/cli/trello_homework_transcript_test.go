package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/asr"
)

func TestIsEnglishHomeworkCard(t *testing.T) {
	cases := []struct {
		name, desc string
		want       bool
	}{
		{"英语：阅读（2026-11-01）", "预计 20 分钟", true},
		{"背单词（第3天）", "作业类型：背单词\n背单词：a,b", true},
		{"阅读（第1天）", "作业类型：阅读英语\n阅读英语链接：/ai/translate?article=1", true},
		{"口算（第37天）", "作业类型：口算", false},
		{"语文：课外班练习册", "类型: 普通作业", false},
	}
	for _, tc := range cases {
		if got := isEnglishHomeworkCard(tc.name, tc.desc); got != tc.want {
			t.Errorf("isEnglishHomeworkCard(%q, %q) = %v, want %v", tc.name, tc.desc, got, tc.want)
		}
	}
}

func TestIsHomeworkAudio(t *testing.T) {
	cases := []struct {
		name, mime string
		want       bool
	}{
		{"homework-audio-abc-1.m4a", "video/mp4", true},
		{"reading.mp3", "audio/mpeg", true},
		{"notes.wav", "", true},
		{"homework.jpg", "image/jpeg", false},
		{"homework.webp", "image/webp", false},
	}
	for _, tc := range cases {
		if got := isHomeworkAudio(tc.name, tc.mime); got != tc.want {
			t.Errorf("isHomeworkAudio(%q, %q) = %v, want %v", tc.name, tc.mime, got, tc.want)
		}
	}
}

type fakeHomeworkTranscriber struct {
	text string
	err  error
}

func (f *fakeHomeworkTranscriber) Transcribe(_ context.Context, _ string, _ []byte) (*asr.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &asr.Result{Text: f.text}, nil
}

func TestTranscribeHomeworkAudioFillsTranscript(t *testing.T) {
	old := homeworkTranscriber
	defer func() { homeworkTranscriber = old }()
	homeworkTranscriber = &fakeHomeworkTranscriber{text: "Robotics Club is cool!"}

	item := trelloHomeworkAttachment{Name: "homework-audio-abc-1.m4a"}
	transcribeHomeworkAudio(context.Background(), &item, []byte("fake-audio"))
	if item.Transcript != "Robotics Club is cool!" || item.TranscriptError != "" {
		t.Fatalf("item = %+v", item)
	}
}

func TestTranscribeHomeworkAudioRecordsFailure(t *testing.T) {
	old := homeworkTranscriber
	defer func() { homeworkTranscriber = old }()
	homeworkTranscriber = &fakeHomeworkTranscriber{err: errors.New("ASR service error: 语音识别服务暂不可用，请稍后再试")}

	item := trelloHomeworkAttachment{Name: "homework-audio-abc-1.m4a"}
	transcribeHomeworkAudio(context.Background(), &item, []byte("fake-audio"))
	if item.Transcript != "" || item.TranscriptError == "" {
		t.Fatalf("item = %+v", item)
	}
}

func TestTranscribeHomeworkAudioSkipsOversized(t *testing.T) {
	old := homeworkTranscriber
	defer func() { homeworkTranscriber = old }()
	homeworkTranscriber = &fakeHomeworkTranscriber{text: "x"}

	item := trelloHomeworkAttachment{Name: "homework-audio-abc-1.m4a"}
	transcribeHomeworkAudio(context.Background(), &item, make([]byte, maxTranscriptAudioBytes+1))
	if item.Transcript != "" || item.TranscriptError == "" {
		t.Fatalf("item = %+v", item)
	}
}

type homeworkASRTransport func(*http.Request) (*http.Response, error)

func (f homeworkASRTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type recordingHomeworkTranscriber struct{ names []string }

func (f *recordingHomeworkTranscriber) Transcribe(_ context.Context, name string, data []byte) (*asr.Result, error) {
	f.names = append(f.names, name)
	if string(data) != "fake-audio" {
		return nil, fmt.Errorf("unexpected audio bytes")
	}
	return &asr.Result{Text: "春眠不觉晓"}, nil
}

func TestHomeworkASRFlagIncludesTranscriptInDownloadedReport(t *testing.T) {
	for _, tc := range []struct {
		name, cardName string
		flags          []string
		want           bool
	}{
		{"all types", "语文朗读", []string{"--asr"}, true},
		{"disabled English", "英语朗读", []string{"--asr=false"}, false},
		{"legacy English default", "英语朗读", nil, true},
		{"legacy non-English default", "语文朗读", nil, false},
		{"legacy off", "英语朗读", []string{"--transcript=false"}, false},
		{"explicit overrides legacy", "语文朗读", []string{"--asr", "--transcript=false"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRELLO_API_KEY", "test-key")
			t.Setenv("TRELLO_TOKEN", "test-token")
			oldTransport, oldASR := http.DefaultTransport, homeworkTranscriber
			t.Cleanup(func() { http.DefaultTransport = oldTransport; homeworkTranscriber = oldASR })
			fake := &recordingHomeworkTranscriber{}
			homeworkTranscriber = fake
			// Exercise the real report/download command; intercept all Trello requests
			// so this test never contacts Trello or an external ASR service.
			http.DefaultTransport = homeworkASRTransport(func(r *http.Request) (*http.Response, error) {
				recorder := httptest.NewRecorder()
				recorder.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/1/boards/board/lists":
					recorder.WriteString(`[{"id":"done","name":"已完成"}]`)
				case "/1/boards/board/cards":
					json.NewEncoder(recorder).Encode([]map[string]any{{"id": "card", "name": tc.cardName, "idList": "done", "due": "2026-11-02T12:00:00Z"}})
				case "/1/cards/card/attachments":
					recorder.WriteString(`[{"id":"audio","name":"reading.m4a","mimeType":"video/mp4"},{"id":"photo","name":"page.jpg","mimeType":"image/jpeg"}]`)
				case "/1/cards/card/attachments/audio":
					recorder.WriteString(`{"id":"audio","name":"reading.m4a","mimeType":"video/mp4"}`)
				case "/1/cards/card/attachments/photo":
					recorder.WriteString(`{"id":"photo","name":"page.jpg","mimeType":"image/jpeg"}`)
				case "/1/cards/card/attachments/audio/download/reading.m4a":
					recorder.WriteString("fake-audio")
				case "/1/cards/card/attachments/photo/download/page.jpg":
					recorder.WriteString("fake-image")
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					recorder.WriteHeader(http.StatusNotFound)
				}
				return recorder.Result(), nil
			})
			var out bytes.Buffer
			cmd := newTrelloHomeworkCommand()
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			args := []string{"--board", "board", "--date", "2026-11-02", "--dir", t.TempDir()}
			cmd.SetArgs(append(args, tc.flags...))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var report trelloHomeworkReport
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if len(report.Cards) != 1 || len(report.Cards[0].Attachments) != 2 {
				t.Fatal(out.String())
			}
			items := report.Cards[0].Attachments
			if items[0].SavedTo == "" || items[0].SizeBytes == 0 {
				t.Fatal("audio not downloaded")
			}
			if tc.want {
				if len(fake.names) != 1 || fake.names[0] != "reading.m4a" || items[0].Transcript != "春眠不觉晓" {
					t.Fatal(out.String(), fake.names)
				}
				if !strings.Contains(formatTrelloHomeworkReport(report), "识别文本: 春眠不觉晓") {
					t.Fatal("pretty report lost transcript")
				}
			} else if len(fake.names) != 0 || items[0].Transcript != "" {
				t.Fatal(out.String(), fake.names)
			}
			if items[1].Transcript != "" {
				t.Fatal("image was transcribed")
			}
		})
	}
}
