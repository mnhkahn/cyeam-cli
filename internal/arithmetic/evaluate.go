package arithmetic

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

var latexFraction = regexp.MustCompile(`\\frac\{([^{}]+)\}\{([^{}]+)\}`)

// NormalizeMath changes typography only. It never guesses handwritten digits.
func NormalizeMath(s string) string {
	// Spotting responses can contain doubly escaped LaTeX and math delimiters.
	for strings.Contains(s, `\\`) {
		s = strings.ReplaceAll(s, `\\`, `\`)
	}
	s = strings.NewReplacer(`\times`, "*", `\cdot`, "*", `\div`, "/", `\left`, "", `\right`, "", `\,`, "", `\;`, "", `\!`, "", `\qquad`, "", `\quad`, "", "$", "", `\(`, "", `\)`, "", `\[`, "", `\]`, "").Replace(s)
	for i := 0; i < 16 && latexFraction.MatchString(s); i++ {
		s = latexFraction.ReplaceAllString(s, "(($1)/($2))")
	}

	s = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	return strings.NewReplacer("＋", "+", "﹢", "+", "－", "-", "−", "-", "﹣", "-", "×", "*", "x", "*", "X", "*", "＊", "*", "÷", "/", "／", "/", "：", "/", "＝", "=", "（", "(", "）", ")", "，", ",").Replace(s)
}

type calculator struct {
	s          string
	pos, depth int
}

// Evaluate uses exact rational arithmetic and a small allowlisted grammar.
// No Go/Python/shell evaluation, identifiers, functions, or implicit multiplication.
func Evaluate(expression string) (*big.Rat, error) {
	s := NormalizeMath(expression)
	if len(s) == 0 || len(s) > 256 {
		return nil, fmt.Errorf("expression must contain 1–256 characters")
	}
	p := calculator{s: s}
	v, err := p.sum()
	if err != nil {
		return nil, err
	}
	if p.pos != len(s) {
		return nil, fmt.Errorf("unsupported expression content")
	}
	return v, nil
}
func (p *calculator) sum() (*big.Rat, error) {
	a, err := p.product()
	if err != nil {
		return nil, err
	}
	for p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
		op := p.s[p.pos]
		p.pos++
		b, err := p.product()
		if err != nil {
			return nil, err
		}
		if op == '+' {
			a.Add(a, b)
		} else {
			a.Sub(a, b)
		}
	}
	return a, nil
}
func (p *calculator) product() (*big.Rat, error) {
	a, err := p.atom()
	if err != nil {
		return nil, err
	}
	for p.pos < len(p.s) && (p.s[p.pos] == '*' || p.s[p.pos] == '/') {
		op := p.s[p.pos]
		p.pos++
		b, err := p.atom()
		if err != nil {
			return nil, err
		}
		if op == '*' {
			a.Mul(a, b)
		} else {
			if b.Sign() == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			a.Quo(a, b)
		}
	}
	return a, nil
}
func (p *calculator) atom() (*big.Rat, error) {
	if p.pos >= len(p.s) {
		return nil, fmt.Errorf("incomplete expression")
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 32 {
		return nil, fmt.Errorf("expression nesting exceeds 32")
	}
	sign := 1
	if p.s[p.pos] == '+' || p.s[p.pos] == '-' {
		if p.s[p.pos] == '-' {
			sign = -1
		}
		p.pos++
		if p.pos >= len(p.s) {
			return nil, fmt.Errorf("incomplete expression")
		}
	}
	var v *big.Rat
	if p.s[p.pos] == '(' {
		p.pos++
		var err error
		v, err = p.sum()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.s) || p.s[p.pos] != ')' {
			return nil, fmt.Errorf("unclosed parentheses")
		}
		p.pos++
	} else {
		start := p.pos
		for p.pos < len(p.s) && ((p.s[p.pos] >= '0' && p.s[p.pos] <= '9') || p.s[p.pos] == '.') {
			p.pos++
		}
		literal := p.s[start:p.pos]
		if len(literal) == 0 || len(literal) > 128 || strings.Count(literal, ".") > 1 || literal == "." {
			return nil, fmt.Errorf("invalid number")
		}
		var ok bool
		v, ok = new(big.Rat).SetString(literal)
		if !ok {
			return nil, fmt.Errorf("invalid number")
		}
	}
	if sign < 0 {
		v.Neg(v)
	}
	return v, nil
}
