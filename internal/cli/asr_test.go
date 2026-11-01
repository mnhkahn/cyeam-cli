package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/asr"
	"github.com/mnhkahn/cyeam-cli/internal/output"
)

type fakeTranscriber struct {
	filename string
	audio    []byte
}

func (f *fakeTranscriber) Transcribe(_ context.Context, filename string, audio []byte) (*asr.Result, error) {
	f.filename = filename
	f.audio = audio
	return &asr.Result{Text: "hello world"}, nil
}

func TestASRCommandWritesJSONEnvelope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reading.m4a")
	if err := os.WriteFile(path, []byte("fake-audio"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	fake := &fakeTranscriber{}
	cmd := NewRootCommand(Dependencies{Stdout: stdout, ASR: fake})
	cmd.SetArgs([]string{"asr", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute asr: %v", err)
	}
	if fake.filename != path || string(fake.audio) != "fake-audio" {
		t.Fatalf("transcriber got file=%q bytes=%q", fake.filename, fake.audio)
	}

	var env output.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal envelope data: %v", err)
	}
	if !bytes.Contains(data, []byte(`"text":"hello world"`)) {
		t.Fatalf("envelope data = %#v", env.Data)
	}
}

func TestASRCommandPrettyPrintsText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reading.m4a")
	if err := os.WriteFile(path, []byte("fake-audio"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Stdout: stdout, ASR: &fakeTranscriber{}})
	cmd.SetArgs([]string{"--pretty", "asr", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute asr --pretty: %v", err)
	}
	if got, want := stdout.String(), "hello world\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestASRCommandMissingFile(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Stdout: stdout, ASR: &fakeTranscriber{}})
	cmd.SetArgs([]string{"asr", filepath.Join(t.TempDir(), "missing.m4a")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing file")
	}
}
