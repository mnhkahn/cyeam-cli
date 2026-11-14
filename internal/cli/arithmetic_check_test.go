package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/homework"
)

const arithmeticResponse = `{"input":{"coverage_uncertain":false,"kind":"arithmetic","count":2,"items":[{"number":1,"expression":"1+1","raw":"2","state":"clear"},{"number":2,"expression":"3*4","raw":"11","state":"clear"}]},"report":{"grade":"差","provisional":false,"correct":1,"incorrect":1,"pending":0,"total":2,"accuracy":50,"items":[{"number":1,"expression":"1+1","raw":"2","state":"clear","status":"correct","hint":"正确"},{"number":2,"expression":"3*4","raw":"11","state":"clear","status":"incorrect","hint":"请重新计算"}],"future_field":"preserved"}}`

func writeHomeworkTestImage(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "homework.png")
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestArithmeticCheckUsesServerThroughRoot(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	path := writeHomeworkTestImage(t)
	for _, pretty := range []bool{false, true} {
		service := &fakeHomeworkService{review: func(_ context.Context, input homework.ReviewRequest) (json.RawMessage, error) {
			if input.Kind != "arithmetic" || input.Count != 2 || input.Filename != path || len(input.Image) == 0 {
				t.Fatalf("unexpected input: %+v", input)
			}
			return json.RawMessage(arithmeticResponse), nil
		}}
		var out bytes.Buffer
		cmd := NewRootCommand(Dependencies{Stdout: &out, Homework: service})
		args := []string{"arithmetic", "check", path, "--expected-count", "2"}
		if pretty {
			args = append(args, "--pretty")
		}
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if pretty {
			for _, want := range []string{"成绩：差", "正确 1，需订正 1，待核对 0 / 共 2 项", "(2) 3*4 = 11  需订正（请重新计算）"} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("missing %q: %s", want, out.String())
				}
			}
		} else {
			var envelope struct {
				OK   bool            `json:"ok"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if !envelope.OK || !strings.Contains(string(envelope.Data), `"future_field":"preserved"`) {
				t.Fatal(out.String())
			}
		}
	}
}

func TestArithmeticCheckRejectsInvalidInputBeforeUpload(t *testing.T) {
	path := writeHomeworkTestImage(t)
	service := &fakeHomeworkService{review: func(context.Context, homework.ReviewRequest) (json.RawMessage, error) {
		t.Fatal("unexpected upload")
		return nil, nil
	}}
	for _, args := range [][]string{{path}, {path, "--expected-count", "0"}, {path, "--expected-count", "101"}} {
		cmd := newArithmeticCheckCommand(service)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatal(args)
		}
	}
	bad := filepath.Join(t.TempDir(), "bad.gif")
	if err := os.WriteFile(bad, []byte("GIF89a"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newArithmeticCheckCommand(service)
	cmd.SetArgs([]string{bad, "--expected-count", "1"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "JPG, PNG, or WebP") {
		t.Fatal(err)
	}
}

func TestArithmeticRegradeSendsSavedInputToServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, []byte(arithmeticResponse), 0600); err != nil {
		t.Fatal(err)
	}
	service := &fakeHomeworkService{regrade: func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
		var value struct {
			Kind  string `json:"kind"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(input, &value); err != nil || value.Kind != "arithmetic" || value.Count != 2 {
			t.Fatalf("input = %s", input)
		}
		return json.RawMessage(arithmeticResponse), nil
	}}
	var out bytes.Buffer
	cmd := newHomeworkRegradeCommand("arithmetic", service)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"future_field":"preserved"`) {
		t.Fatal(out.String())
	}
}
