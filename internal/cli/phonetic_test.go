package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/phonetic"
)

type fakePhoneticFetcher struct {
	word string
}

func (f *fakePhoneticFetcher) Fetch(_ context.Context, word string) (*phonetic.Result, error) {
	f.word = word
	return &phonetic.Result{Word: word, UKPhonetic: "/test/", Definitions: []string{"a test"}}, nil
}

func TestPhoneticCommandWritesJSONEnvelope(t *testing.T) {
	stdout := new(bytes.Buffer)
	fetcher := &fakePhoneticFetcher{}
	cmd := NewRootCommand(Dependencies{Stdout: stdout, Phonetic: fetcher})
	cmd.SetArgs([]string{"phonetic", "C++"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute phonetic: %v", err)
	}
	if fetcher.word != "C++" {
		t.Fatalf("word = %q", fetcher.word)
	}

	var env output.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	// 命令输出是合法 JSON 时信封原样嵌套，data 是对象而不是二次编码的字符串。
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal envelope data: %v", err)
	}
	if !bytes.Contains(data, []byte(`"word":"C++"`)) {
		t.Fatalf("envelope data = %#v", env.Data)
	}
}

func TestPhoneticCommandPrettyUsesCommandOutput(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Stdout: stdout, Phonetic: &fakePhoneticFetcher{}})
	cmd.SetArgs([]string{"--pretty", "phonetic", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute phonetic --pretty: %v", err)
	}
	if got, want := stdout.String(), "单词: hello\n英式: test\n释义:\n  1. a test\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
