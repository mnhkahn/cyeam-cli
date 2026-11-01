package arithmetic

import (
	"go/constant"
	"go/token"
	"go/types"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var digits = regexp.MustCompile(`[0-9]+`)

func numbers(s string) []int {
	out := []int{}
	for _, s := range digits.FindAllString(s, -1) {
		v, _ := strconv.Atoi(s)
		out = append(out, v)
	}
	return out
}
func eval(t *testing.T, s string) int {
	t.Helper()
	s = strings.NewReplacer("＋", "+", "－", "-", "×", "*", "÷", "/", "（", "(", "）", ")").Replace(s)
	v, err := types.Eval(token.NewFileSet(), nil, 0, s)
	if err != nil {
		t.Fatalf("evaluate %q: %v", s, err)
	}
	n, ok := constant.Int64Val(constant.ToInt(v.Value))
	if !ok {
		t.Fatalf("noninteger %s", s)
	}
	return int(n)
}
func TestNumericAnswersAndCurriculumBounds(t *testing.T) {
	for _, typ := range Types(0) {
		t.Run(typ.ID, func(t *testing.T) {
			r := rand.New(rand.NewSource(42))
			for i := 0; i < 1000; i++ {
				q := candidate(r, typ.ID)
				ns := numbers(q.Text)
				ans := numbers(q.Answer)
				if q.Text == "" || q.Answer == "" {
					t.Fatal("empty question")
				}
				text := strings.TrimPrefix(q.Text, "列竖式计算：")
				// Independent Go constant evaluator checks normal expressions, including precedence.
				if strings.Contains(text, "＝") && !strings.Contains(text, "已知") && typ.ID != "money" && typ.ID != "length" && typ.ID != "remainder" {
					expression := strings.Split(text, "＝")[0]
					if got := eval(t, expression); len(ans) != 1 || got != ans[0] {
						t.Fatalf("%s answer %s, computed %d", q.Text, q.Answer, got)
					}
				}
				if typ.Grade == 1 && typ.ID != "money" || strings.HasPrefix(typ.ID, "column-") || strings.HasPrefix(typ.ID, "three-digit-") {
					limit := 100
					if strings.HasPrefix(typ.ID, "three-digit-") {
						limit = 999
					}
					expr := strings.Split(text, "＝")[0]
					parts := strings.FieldsFunc(expr, func(r rune) bool { return r == '＋' || r == '－' })
					prefix := parts[0]
					if v := eval(t, prefix); v < 0 || v > limit {
						t.Fatal(q)
					}
					pos := len([]rune(parts[0]))
					rs := []rune(expr)
					for _, part := range parts[1:] {
						prefix += string(rs[pos]) + part
						pos += 1 + len([]rune(part))
						if v := eval(t, prefix); v < 0 || v > limit {
							t.Fatal(q)
						}
					}
				}
				switch typ.ID {
				case "add", "subtract", "column-add", "column-subtract":
					if ns[0] < 10 || ns[1] < 10 || ns[0] > 99 || ns[1] > 99 {
						t.Fatal(q)
					}
				case "multiply":
					if ns[0] < 1 || ns[0] > 9 || ns[1] < 1 || ns[1] > 9 {
						t.Fatal(q)
					}
				case "divide":
					if ns[0]%ns[1] != 0 || ns[1] > 9 || ans[0] > 9 {
						t.Fatal(q)
					}
				case "mixed-multiply-divide":
					if !strings.Contains(text, "×") || !strings.Contains(text, "÷") {
						t.Fatal(q)
					}
				case "remainder":
					if ns[1] < 2 || ns[1] > 9 || ans[1] <= 0 || ans[1] >= ns[1] || ans[0]*ns[1]+ans[1] != ns[0] {
						t.Fatal(q)
					}
				case "multiply-subtract":
					if ans[0] < 0 {
						t.Fatal(q)
					}
				case "round-multiply":
					if ns[0]%10 != 0 && ns[1]%10 != 0 {
						t.Fatal(q)
					}
				case "product-change":
					if ns[0]*ns[1] != ns[2] || ns[3]*ns[4] != ans[0] {
						t.Fatal(q)
					}
				case "split-multiply":
					if ns[0] < 11 || ns[0] > 25 || ns[1] < 11 || ns[1] > 15 {
						t.Fatal(q)
					}
				case "mult-3x2":
					x, y := ns[0], ns[1]
					if x < 100 {
						x, y = y, x
					}
					if x < 100 || x > 999 || y < 11 || y > 99 {
						t.Fatal(q)
					}
				case "round-3x2":
					x, y := ns[0], ns[1]
					if x < 100 {
						x, y = y, x
					}
					if x < 100 || x > 999 || y < 10 || y > 99 {
						t.Fatal(q)
					}
					if x%10 != 0 || y%10 != 0 {
						t.Fatal(q)
					}
				case "add-laws":
					if ns[0]+ns[2] != 100 {
						t.Fatal(q)
					}
				case "multiply-laws":
					if ns[0]*ns[2] != 100 && ns[0]*ns[2] != 1000 {
						t.Fatal(q)
					}
				case "money":
					got := ans[0]
					if len(ans) == 2 {
						got = ans[0]*10 + ans[1]
					}
					want := ns[0]*10 + ns[1]
					if strings.Contains(text, "－") {
						want = ns[0]*10 - ns[1]
					}
					if got != want || !strings.ContainsAny(q.Answer, "元角分") {
						t.Fatal(q)
					}
				case "length":
					got := ans[0]
					if len(ans) == 2 {
						got = ans[0]*100 + ans[1]
					}
					want := ns[0]
					if len(ns) > 1 {
						want = ns[0]*100 + ns[1]
					}
					if strings.Contains(text, "－") {
						want = ns[0]*100 - ns[1]
					}
					if got != want || !strings.Contains(q.Answer, "米") {
						t.Fatal(q)
					}
				case "large-number":
					scale := 10000
					if strings.Contains(text, "亿") {
						scale = 100000000
					}
					switch {
					case strings.Contains(text, "≈"):
						if ans[0] != (ns[0]+scale/2)/scale {
							t.Fatal(q)
						}
					case strings.Contains(text, "里面"):
						if ans[0]*scale != ns[0] {
							t.Fatal(q)
						}
					case strings.Contains(text, "写作"):
						if ns[0]*scale != ans[0] {
							t.Fatal(q)
						}
					default:
						if ns[1]*scale+ans[0]*1000 != ns[0] {
							t.Fatal(q)
						}
					}
				case "quantity":
					want := ns[0] * ns[1]
					if strings.Contains(text, "相遇") {
						want = (ns[0] + ns[1]) * ns[2]
					} else if strings.Contains(text, "每本（") {
						want = ns[1] / ns[0]
					}
					if ans[0] != want {
						t.Fatal(q)
					}
				case "estimate":
					round := 10
					if strings.Contains(text, "整百") {
						round = 100
					}
					want := ((ns[0] + round/2) / round * round) * ((ns[1] + 5) / 10 * 10)
					if ans[0] != want || !strings.Contains(text, "四舍五入") {
						t.Fatal(q)
					}
				}
			}
		})
	}
}

func TestGenerateBalanceUniquenessAndAnswers(t *testing.T) {
	for _, grade := range []int{1, 2, 4} {
		opts := Options{Grade: grade, Count: 60, Seed: 123, Answers: true}
		e, err := Generate(opts)
		if err != nil {
			t.Fatal(err)
		}
		again, _ := Generate(opts)
		if !reflect.DeepEqual(e, again) {
			t.Fatal("seed is not reproducible")
		}
		seen := map[string]bool{}
		counts := map[string]int{}
		for _, q := range e.Questions {
			if seen[q.Text] || q.Answer == "" {
				t.Fatal(q)
			}
			seen[q.Text] = true
			counts[q.Type]++
		}
		for _, typ := range Types(grade) {
			if counts[typ.ID] < 60/len(Types(grade)) || counts[typ.ID] > (60+len(Types(grade))-1)/len(Types(grade)) {
				t.Fatal(counts)
			}
		}
		opts.Answers = false
		hidden, _ := Generate(opts)
		for i, q := range hidden.Questions {
			if q.Answer != "" || q.Text != e.Questions[i].Text {
				t.Fatal("answer suppression changed questions")
			}
		}
	}
}
func TestValidationAndExhaustion(t *testing.T) {
	for _, o := range []Options{{}, {Grade: 3}, {Grade: 1, Count: -1}, {Grade: 1, Count: 1001}, {Types: []string{"unknown"}}, {Grade: 1, Types: []string{"multiply"}}, {Types: []string{"multiply", "add"}}, {Types: []string{"multiply"}, Count: 82}} {
		if _, err := Generate(o); err == nil {
			t.Fatalf("expected error: %+v", o)
		}
	}
	e, err := Generate(Options{Types: []string{"表内乘法"}, Seed: 0})
	if err != nil || e.Grade != 2 || len(e.Questions) != 20 {
		t.Fatalf("%+v %v", e, err)
	}
	e, err = Generate(Options{Types: []string{"multiply", "multiply"}, Count: 81})
	if err != nil || len(e.Questions) != 81 {
		t.Fatalf("full table: %v", err)
	}
}
