package cli

import (
	"context"
	"errors"
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
