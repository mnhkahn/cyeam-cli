package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
	"github.com/mnhkahn/cyeam-cli/internal/client"
	"github.com/mnhkahn/cyeam-cli/internal/cyeam"
)

func newArithmeticTestService(t *testing.T) *cyeam.Service {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/arithmetic/types":
			types := []arithmetic.Type{{ID: "multiply", Name: "表内乘法", Grade: 2}, {ID: "column-mixed", Name: "100以内竖式连加减", Grade: 2}}
			if r.URL.Query().Get("grade") == "4" {
				types = []arithmetic.Type{{ID: "server-added", Name: "服务端新增题型", Grade: 4}}
			}
			_ = json.NewEncoder(w).Encode(types)
		case "/api/arithmetic/generate":
			count, _ := strconv.Atoi(r.URL.Query().Get("count"))
			seed, _ := strconv.ParseInt(r.URL.Query().Get("seed"), 10, 64)
			exercise := arithmetic.Exercise{Grade: 2, Seed: seed, Questions: []arithmetic.Question{}}
			for i := 0; i < count; i++ {
				question := arithmetic.Question{Type: r.URL.Query().Get("type"), Text: fmt.Sprintf("%d×1＝（ ）", i+1)}
				if r.URL.Query().Get("answers") == "true" {
					question.Answer = strconv.Itoa(i + 1)
				}
				exercise.Questions = append(exercise.Questions, question)
			}
			_ = json.NewEncoder(w).Encode(exercise)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return cyeam.NewService(client.New(server.URL, server.Client()), server.URL)
}

func TestArithmeticRootJSONAndText(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	service := newArithmeticTestService(t)
	var out bytes.Buffer
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: &out})
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
	for _, question := range envelope.Data.Questions {
		if question.Answer != "" {
			t.Fatal("answers exposed by default")
		}
	}
	for _, answers := range []bool{false, true} {
		out.Reset()
		cmd = NewRootCommand(Dependencies{Service: service, Stdout: &out})
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
	service := newArithmeticTestService(t)
	var out bytes.Buffer
	cmd := newArithmeticCommand(Dependencies{Service: service})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"types", "--grade", "4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var types []arithmetic.Type
	if err := json.Unmarshal(out.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].ID != "server-added" {
		t.Fatal(types)
	}
	for _, args := range [][]string{{"--grade", "-1"}, {"--type", "multiply", "--count", "0"}, {"--grade", "1", "--format", "pdf"}, {"types", "--grade", "-1"}, {"unexpected"}} {
		cmd = newArithmeticCommand(Dependencies{Service: service})
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
	exercise := arithmetic.Exercise{
		Grade: 2,
		Types: []arithmetic.Type{{ID: "column-mixed", Name: "100以内竖式连加减", Grade: 2}},
		Questions: []arithmetic.Question{
			{Type: "column-mixed", Text: "列竖式计算：12＋13－5＝（ ）", Answer: "20"},
			{Type: "column-mixed", Text: "列竖式计算：22＋13－5＝（ ）", Answer: "30"},
		},
	}
	if err := writeArithmetic(&out, exercise, "worksheet"); err != nil {
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
