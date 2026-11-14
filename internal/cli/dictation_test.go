package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/homework"
)

func TestDictationCheckUsesServer(t *testing.T) {
	image := writeHomeworkTestImage(t)
	words := filepath.Join(t.TempDir(), "words.txt")
	if err := os.WriteFile(words, []byte("breakfast，biscuit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	service := &fakeHomeworkService{review: func(_ context.Context, input homework.ReviewRequest) (json.RawMessage, error) {
		if input.Kind != "dictation" || input.Count != 2 || len(input.Words) != 2 || input.Words[0] != "breakfast" || input.Words[1] != "biscuit" {
			t.Fatalf("input = %+v", input)
		}
		return json.RawMessage(`{"input":{"kind":"dictation","count":2,"words":["breakfast","biscuit"],"items":[]},"report":{"grade":"优","provisional":true,"correct":1,"incorrect":0,"pending":1,"total":2,"accuracy":100,"items":[{"number":1,"raw":"breakfast","state":"clear","status":"correct","hint":"正确"},{"number":2,"raw":"","state":"uncertain","status":"pending","hint":"请对照原图核对字迹"}]}}`), nil
	}}
	var out bytes.Buffer
	cmd := newDictationCommand(service)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"check", image, "--words", words})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"kind":"dictation"`, `"grade":"优"`, `"pending":1`} {
		if !strings.Contains(out.String(), want) {
			t.Fatal(out.String())
		}
	}
}

func TestDictationRejectsInvalidWords(t *testing.T) {
	image := writeHomeworkTestImage(t)
	service := &fakeHomeworkService{review: func(context.Context, homework.ReviewRequest) (json.RawMessage, error) {
		t.Fatal("unexpected upload")
		return nil, nil
	}}
	for _, content := range []string{"", strings.Repeat("a\n", 101)} {
		words := filepath.Join(t.TempDir(), "words.txt")
		if err := os.WriteFile(words, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := newDictationCommand(service)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"check", image, "--words", words})
		if err := cmd.Execute(); err == nil {
			t.Fatal("accepted invalid words")
		}
	}
}
