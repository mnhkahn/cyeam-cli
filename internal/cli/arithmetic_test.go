package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
)

func TestArithmeticRootJSONAndText(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	var out bytes.Buffer
	cmd := NewRootCommand(Dependencies{Stdout: &out})
	cmd.SetArgs([]string{"arithmetic", "--type", "multiply", "--seed", "0"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		OK   bool                `json:"ok"`
		Data arithmetic.Exercise `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || len(envelope.Data.Questions) != 20 || envelope.Data.Grade != 2 {
		t.Fatal(out.String())
	}
	for _, q := range envelope.Data.Questions {
		if q.Answer != "" {
			t.Fatal("answers exposed by default")
		}
	}
	for _, answers := range []bool{false, true} {
		out.Reset()
		cmd = NewRootCommand(Dependencies{Stdout: &out})
		args := []string{"arithmetic", "--type", "multiply", "--seed", "0", "--format", "text", "--pretty"}
		if answers {
			args = append(args, "--answers")
		}
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		questions, solution, _ := strings.Cut(out.String(), "\n答案\n")
		lines := strings.Split(strings.TrimSpace(questions), "\n")
		if len(lines) != 20 || lines[0] != envelope.Data.Questions[0].Text {
			t.Fatal(out.String())
		}
		if answers && solution == "" || !answers && solution != "" {
			t.Fatal(out.String())
		}
	}
}
func TestArithmeticTypesAndInvalidFlags(t *testing.T) {
	var out bytes.Buffer
	cmd := newArithmeticCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"types", "--grade", "4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var types []arithmetic.Type
	if err := json.Unmarshal(out.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if len(types) != 11 {
		t.Fatal(types)
	}
	for _, args := range [][]string{{"--grade", "3"}, {"--type", "multiply", "--count", "0"}, {"--grade", "1", "--format", "pdf"}, {"types", "--grade", "3"}, {"unexpected"}} {
		cmd = newArithmeticCommand()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}
func TestArithmeticWorksheet(t *testing.T) {
	var out bytes.Buffer
	e, err := arithmetic.Generate(arithmetic.Options{Types: []string{"column-mixed"}, Count: 2, Answers: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArithmetic(&out, e, "worksheet"); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"北师大版口算练习", "姓名：", "2年级｜100以内竖式连加减｜2题", "1.", "2.", "──────", "\n答案\n"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing %q: %s", text, out.String())
		}
	}
	if strings.Contains(strings.Split(out.String(), "\n答案\n")[0], "＝") {
		t.Fatal("vertical worksheet should leave answer line blank")
	}
}
