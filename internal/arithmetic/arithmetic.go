// Package arithmetic generates local, curriculum-bounded oral arithmetic exercises.
package arithmetic

import (
	"fmt"
	"math/rand"
	"strings"
)

type Type struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Grade int    `json:"grade"`
}

var catalog = []Type{
	{"add", "100以内加法", 1}, {"subtract", "100以内减法", 1},
	{"chain-add", "连加", 1}, {"chain-subtract", "连减", 1}, {"mixed-add-subtract", "加减混合", 1}, {"money", "人民币计算", 1},
	{"column-add", "100以内竖式加法", 2}, {"column-subtract", "100以内竖式减法", 2}, {"column-mixed", "100以内竖式连加减", 2},
	{"multiply", "表内乘法", 2}, {"multiply-add", "乘加", 2}, {"multiply-subtract", "乘减", 2},
	{"divide", "表内除法", 2}, {"mixed-multiply-divide", "乘除混合", 2}, {"remainder", "有余数除法", 2},
	{"three-digit-add", "三位数加法", 2}, {"three-digit-subtract", "三位数减法", 2}, {"three-digit-chain", "三位数连加减", 2}, {"length", "长度单位换算", 2},
	{"round-multiply", "整十整百整千乘法", 4}, {"product-change", "积的变化规律", 4}, {"split-multiply", "两位数乘两位数拆分", 4},
	{"add-laws", "加法交换律结合律", 4}, {"multiply-laws", "乘法交换律结合律", 4},
	{"distribute", "乘法分配律", 4}, {"large-number", "大数读写与填空", 4},
	{"quantity", "数量关系", 4}, {"estimate", "乘法估算", 4},
}

func Types(grade int) []Type {
	result := []Type{}
	for _, t := range catalog {
		if grade == 0 || t.Grade == grade {
			result = append(result, t)
		}
	}
	return result
}

type Options struct {
	Grade, Count int
	Types        []string
	Seed         int64
	Answers      bool
}
type Question struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Answer string `json:"answer,omitempty"`
}
type Exercise struct {
	Grade     int        `json:"grade"`
	Seed      int64      `json:"seed"`
	Questions []Question `json:"questions"`
}

// Generate defaults to 20 questions. Grade can be inferred only from named types.
// Mixed exercises distribute questions evenly across the requested types.
func Generate(o Options) (Exercise, error) {
	result := Exercise{Seed: o.Seed}
	if o.Count == 0 {
		o.Count = 20
	}
	if o.Count < 1 || o.Count > 1000 {
		return result, fmt.Errorf("count must be between 1 and 1000")
	}
	if o.Grade != 0 && o.Grade != 1 && o.Grade != 2 && o.Grade != 4 {
		return result, fmt.Errorf("supported grades: 1, 2, 4 (2026 first semester)")
	}
	selected := []Type{}
	seenTypes := map[string]bool{}
	for _, name := range o.Types {
		name = strings.TrimSpace(name)
		var found *Type
		for i := range catalog {
			if catalog[i].ID == name || catalog[i].Name == name {
				found = &catalog[i]
				break
			}
		}
		if found == nil {
			return result, fmt.Errorf("unknown arithmetic type %q; use arithmetic types", name)
		}
		if o.Grade == 0 {
			o.Grade = found.Grade
		}
		if found.Grade != o.Grade {
			return result, fmt.Errorf("type %s belongs to grade %d, not grade %d", name, found.Grade, o.Grade)
		}
		if !seenTypes[found.ID] {
			selected = append(selected, *found)
			seenTypes[found.ID] = true
		}
	}
	if o.Grade == 0 {
		return result, fmt.Errorf("--grade is required for mixed practice; or select a specific --type")
	}
	if len(selected) == 0 {
		selected = Types(o.Grade)
	}
	result.Grade = o.Grade
	r := rand.New(rand.NewSource(o.Seed))
	r.Shuffle(len(selected), func(i, j int) { selected[i], selected[j] = selected[j], selected[i] })
	seen := map[string]bool{}
	for i := 0; i < o.Count; i++ {
		typ := selected[i%len(selected)]
		var q Question
		found := false
		for attempt := 0; attempt < 10000; attempt++ {
			q = candidate(r, typ.ID)
			if !seen[q.Text] {
				found = true
				break
			}
		}
		if !found {
			return Exercise{}, fmt.Errorf("cannot generate %d distinct questions for %s; reduce --count or add types", o.Count, typ.ID)
		}
		seen[q.Text] = true
		q.Type = typ.ID
		if !o.Answers {
			q.Answer = ""
		}
		result.Questions = append(result.Questions, q)
	}
	return result, nil
}

func candidate(r *rand.Rand, kind string) Question {
	n := func(lo, hi int) int { return lo + r.Intn(hi-lo+1) }
	q := func(text string, answer int) Question { return Question{Text: text, Answer: fmt.Sprint(answer)} }
	expr := func(a int, op string, b, answer int) Question {
		return q(fmt.Sprintf("%d%s%d＝（ ）", a, op, b), answer)
	}
	a, b, c := n(1, 9), n(1, 9), n(1, 9)
	switch kind {
	case "add", "column-add":
		a = n(10, 90)
		b = n(10, 100-a)
		v := expr(a, "＋", b, a+b)
		if kind == "column-add" {
			v.Text = "列竖式计算：" + v.Text
		}
		return v
	case "subtract", "column-subtract":
		a = n(10, 99)
		b = n(10, a)
		v := expr(a, "－", b, a-b)
		if kind == "column-subtract" {
			v.Text = "列竖式计算：" + v.Text
		}
		return v
	case "chain-add":
		a = n(10, 70)
		b = n(10, 90-a)
		c = n(1, 100-a-b)
		return q(fmt.Sprintf("%d＋%d＋%d＝（ ）", a, b, c), a+b+c)
	case "chain-subtract":
		a = n(30, 100)
		b = n(10, a-10)
		c = n(1, a-b)
		return q(fmt.Sprintf("%d－%d－%d＝（ ）", a, b, c), a-b-c)
	case "mixed-add-subtract", "column-mixed":
		a = n(10, 70)
		b = n(10, 100-a)
		c = n(1, a+b)
		v := q(fmt.Sprintf("%d＋%d－%d＝（ ）", a, b, c), a+b-c)
		if kind == "column-mixed" {
			v.Text = "列竖式计算：" + v.Text
		}
		return v
	case "money":
		switch n(0, 3) {
		case 0:
			return Question{Text: fmt.Sprintf("%d元%d角＝（ ）角", a, b), Answer: fmt.Sprintf("%d角", a*10+b)}
		case 1:
			return Question{Text: fmt.Sprintf("%d角%d分＝（ ）分", a, b), Answer: fmt.Sprintf("%d分", a*10+b)}
		case 2:
			return Question{Text: fmt.Sprintf("%d元＋%d角＝（ ）元（ ）角", a, b), Answer: fmt.Sprintf("%d元%d角", a, b)}
		default:
			return Question{Text: fmt.Sprintf("%d元－%d角＝（ ）元（ ）角", a, b), Answer: fmt.Sprintf("%d元%d角", a-1, 10-b)}
		}
	case "multiply":
		return expr(a, "×", b, a*b)
	case "multiply-add":
		return q(fmt.Sprintf("%d×%d＋%d＝（ ）", a, b, c), a*b+c)
	case "multiply-subtract":
		c = n(1, min(9, a*b))
		return q(fmt.Sprintf("%d×%d－%d＝（ ）", a, b, c), a*b-c)
	case "divide":
		return expr(a*b, "÷", b, a)
	case "mixed-multiply-divide":
		// Both operations stay within the multiplication table, with exact division.
		divisors := []int{}
		for d := 1; d <= 9; d++ {
			if a*b%d == 0 && a*b/d <= 9 {
				divisors = append(divisors, d)
			}
		}
		c = divisors[r.Intn(len(divisors))]
		if n(0, 1) == 0 {
			return q(fmt.Sprintf("%d×%d÷%d＝（ ）", a, b, c), a*b/c)
		}
		return q(fmt.Sprintf("%d÷%d×%d＝（ ）", a*b, b, c), a*c)
	case "remainder":
		b = n(2, 9)
		c = n(1, b-1)
		return Question{Text: fmt.Sprintf("%d÷%d＝（ ）……（ ）", a*b+c, b), Answer: fmt.Sprintf("%d……%d", a, c)}
	case "three-digit-add":
		a = n(100, 899)
		b = n(100, 999-a)
		return expr(a, "＋", b, a+b)
	case "three-digit-subtract":
		a = n(100, 999)
		b = n(100, a)
		return expr(a, "－", b, a-b)
	case "three-digit-chain":
		a = n(100, 699)
		b = n(100, 899-a)
		c = n(100, a+b)
		if n(0, 1) == 0 {
			c = n(100, 999-a-b)
			return q(fmt.Sprintf("%d＋%d＋%d＝（ ）", a, b, c), a+b+c)
		}
		return q(fmt.Sprintf("%d＋%d－%d＝（ ）", a, b, c), a+b-c)
	case "length":
		b = n(1, 99)
		switch n(0, 2) {
		case 0:
			return Question{Text: fmt.Sprintf("%d米%d厘米＝（ ）厘米", a, b), Answer: fmt.Sprintf("%d厘米", a*100+b)}
		case 1:
			return Question{Text: fmt.Sprintf("%d厘米＝（ ）米（ ）厘米", a*100+b), Answer: fmt.Sprintf("%d米%d厘米", a, b)}
		default:
			return Question{Text: fmt.Sprintf("%d米－%d厘米＝（ ）厘米", a, b), Answer: fmt.Sprintf("%d厘米", a*100-b)}
		}
	case "round-multiply":
		a = n(2, 30)
		b = n(1, 9) * []int{10, 100, 1000}[n(0, 2)]
		if n(0, 1) == 0 {
			a *= 10
		}
		return expr(a, "×", b, a*b)
	case "product-change":
		a = n(11, 19)
		b = n(2, 5)
		x, y := []int{1, 10, 100}[n(0, 2)], []int{10, 100}[n(0, 1)]
		return q(fmt.Sprintf("已知%d×%d＝%d，%d×%d＝（ ）", a, b, a*b, a*x, b*y), a*b*x*y)
	case "split-multiply":
		a = n(11, 25)
		b = n(11, 15)
		return expr(a, "×", b, a*b)
	case "add-laws":
		a = n(11, 89)
		b = n(11, 199)
		c = 100 - a
		return q(fmt.Sprintf("%d＋%d＋%d＝（ ）", a, b, c), a+b+c)
	case "multiply-laws":
		pairs := [][2]int{{25, 4}, {125, 8}, {50, 2}}
		pair := pairs[n(0, 2)]
		b = n(3, 29)
		return q(fmt.Sprintf("%d×%d×%d＝（ ）", pair[0], b, pair[1]), pair[0]*b*pair[1])
	case "distribute":
		a = n(11, 89)
		b = []int{5, 25, 50, 125}[n(0, 3)]
		switch n(0, 2) {
		case 0:
			c = n(1, 9)
			a = n(2, 8) * 10
			return q(fmt.Sprintf("（%d＋%d）×%d＝（ ）", a, c, b), (a+c)*b)
		case 1:
			return q(fmt.Sprintf("%d×%d＋%d×%d＝（ ）", a, b, 100-a, b), 100*b)
		default:
			a = n(2, 9) * 10
			c = n(1, 9)
			return q(fmt.Sprintf("%d×（%d－%d）＝（ ）", b, a, c), b*(a-c))
		}
	case "large-number":
		unit, scale := "万", 10000
		if n(0, 1) == 0 {
			unit, scale = "亿", 100000000
		}
		a = n(10, 999)
		switch n(0, 3) {
		case 0:
			return q(fmt.Sprintf("%d里面有（ ）个%s", a*scale, unit), a)
		case 1:
			return Question{Text: fmt.Sprintf("%d个%s写作（ ）", a, unit), Answer: fmt.Sprint(a * scale)}
		case 2:
			b = n(1, 9)
			return Question{Text: fmt.Sprintf("%d≈（ ）%s（四舍五入到%s位）", a*scale+b*(scale/10), unit, unit), Answer: fmt.Sprintf("%d%s", a+boolInt(b >= 5), unit)}
		default:
			b = n(1, 9)
			return q(fmt.Sprintf("%d由%d个%s和（ ）个千组成", a*scale+b*1000, a, unit), b)
		}
	case "quantity":
		a = n(2, 9)
		b = n(2, 9)
		switch n(0, 3) {
		case 0:
			return Question{Text: fmt.Sprintf("每本书%d元，买%d本，一共（ ）元。", a, b), Answer: fmt.Sprintf("%d元", a*b)}
		case 1:
			return Question{Text: fmt.Sprintf("每小时行%d千米，行驶%d小时，共行（ ）千米。", a*10, b), Answer: fmt.Sprintf("%d千米", a*10*b)}
		case 2:
			return Question{Text: fmt.Sprintf("两人同时从两地相向而行，速度分别为每分钟%d米和%d米，%d分钟相遇，两地相距（ ）米。", a*10, b*10, c), Answer: fmt.Sprintf("%d米", (a+b)*10*c)}
		default:
			return Question{Text: fmt.Sprintf("%d本书共%d元，每本（ ）元。", a, a*b), Answer: fmt.Sprintf("%d元", b)}
		}
	case "estimate":
		a = n(11, 39)
		b = n(11, 39)
		if n(0, 1) == 0 {
			a = n(101, 349)
			rounded := (a + 50) / 100 * 100
			return q(fmt.Sprintf("%d×%d≈（ ）（分别四舍五入到整百、整十）", a, b), rounded*((b+5)/10*10))
		}
		return q(fmt.Sprintf("%d×%d≈（ ）（两个乘数均四舍五入到整十）", a, b), ((a+5)/10*10)*((b+5)/10*10))
	}
	panic("unregistered arithmetic type: " + kind)
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
