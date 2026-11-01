package arithmetic

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

type CheckOptions struct {
	MinConfidence float64
	ExpectedCount int
}
type CheckedQuestion struct {
	Number          int    `json:"number,omitempty"`
	Expression      string `json:"expression"`
	SubmittedAnswer string `json:"submitted_answer,omitempty"`
	ExpectedAnswer  string `json:"expected_answer,omitempty"`
	// Comparison records the calculation; status also accounts for OCR evidence.
	Comparison string        `json:"comparison,omitempty"`
	Difference string        `json:"difference,omitempty"`
	Status     string        `json:"status"`
	Reason     string        `json:"reason,omitempty"`
	Confidence *float64      `json:"confidence,omitempty"`
	Page       int           `json:"page"`
	Box        paddleocr.Box `json:"box"`
	Source     []int         `json:"source_lines"`
}
type CheckReport struct {
	Engine            string            `json:"engine"`
	Detected          int               `json:"detected"`
	Correct           int               `json:"correct"`
	Incorrect         int               `json:"incorrect"`
	NeedsReview       int               `json:"needs_review"`
	MatchingValues    int               `json:"matching_values"`
	MismatchingValues int               `json:"mismatching_values"`
	ExpectedCount     int               `json:"expected_count,omitempty"`
	MissingNumbers    []int             `json:"missing_numbers,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	Questions         []CheckedQuestion `json:"questions"`
	Lines             []paddleocr.Line  `json:"ocr_lines"`
}

var questionLabel = regexp.MustCompile(`^\(?([0-9]{1,4})\)`)
var standaloneLabel = regexp.MustCompile(`^\(([0-9]{1,4})\)$`)
var numericAnswer = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(?:/[0-9]+)?$`)
var remainderAnswer = regexp.MustCompile(`^([0-9]+)(?:……|…|\.{3,})([0-9]+)$`)

func splitEquation(raw string) (number int, expression, answer string, ok bool) {
	text := NormalizeMath(raw)
	if match := questionLabel.FindStringSubmatch(text); match != nil {
		rest := text[len(match[0]):]
		if len(rest) > 0 && ((rest[0] >= '0' && rest[0] <= '9') || rest[0] == '(') {
			number, _ = strconv.Atoi(match[1])
			text = rest
		}
	}
	left, right, hasEquals := strings.Cut(text, "=")
	if !hasEquals || !strings.ContainsAny(left, "+-*/") {
		return 0, "", "", false
	}
	return number, left, right, true
}
func answerValue(answer string) (*big.Rat, error) {
	answer = NormalizeMath(answer)
	if !numericAnswer.MatchString(answer) {
		return nil, fmt.Errorf("answer is missing or not an unambiguous number")
	}
	return Evaluate(answer)
}
func comparison(expression, answer string) (expected string, equal bool, err error) {
	value, err := Evaluate(expression)
	if err != nil {
		return "", false, err
	}
	expected = value.RatString()
	answer = NormalizeMath(answer)
	if m := remainderAnswer.FindStringSubmatch(answer); m != nil {
		parts := strings.Split(expression, "/")
		if len(parts) != 2 {
			return expected, false, fmt.Errorf("remainder answer requires a single integer division")
		}
		a, e1 := Evaluate(parts[0])
		b, e2 := Evaluate(parts[1])
		if e1 != nil || e2 != nil || !a.IsInt() || !b.IsInt() || a.Sign() < 0 || b.Sign() <= 0 {
			return expected, false, fmt.Errorf("unsupported remainder expression")
		}
		quotient, rem := new(big.Int), new(big.Int)
		quotient.QuoRem(a.Num(), b.Num(), rem)
		expected = quotient.String() + "……" + rem.String()
		submittedQ, _ := new(big.Int).SetString(m[1], 10)
		submittedR, _ := new(big.Int).SetString(m[2], 10)
		return expected, submittedQ.Cmp(quotient) == 0 && submittedR.Cmp(rem) == 0, nil
	}
	submitted, err := answerValue(answer)
	if err != nil {
		return expected, false, err
	}
	return expected, value.Cmp(submitted) == 0, nil
}
func sameRow(a, b paddleocr.Box) bool {
	if !a.Valid() || !b.Valid() {
		return false
	}
	height := max(a[3]-a[1], b[3]-b[1])
	return math.Abs((a[1]+a[3]-b[1]-b[3])/2) <= height*0.75
}
func combineConfidence(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := min(*a, *b)
	return &v
}

// Check only pairs equation anchors with same-row answers to their right.
// Scratch work below an equation is never used as its submitted answer.
func Check(lines []paddleocr.Line, o CheckOptions) (CheckReport, error) {
	report := CheckReport{Engine: "paddleocr", ExpectedCount: o.ExpectedCount, Questions: []CheckedQuestion{}, Lines: lines}
	if math.IsNaN(o.MinConfidence) || o.MinConfidence < 0 || o.MinConfidence > 1 {
		return report, fmt.Errorf("minimum confidence must be between 0 and 1")
	}
	if o.ExpectedCount < 0 || o.ExpectedCount > 1000 {
		return report, fmt.Errorf("expected count must be between 1 and 1000 when supplied")
	}
	anchors := map[int]bool{}
	for i, line := range lines {
		if _, _, _, ok := splitEquation(line.Text); ok {
			anchors[i] = true
		}
	}
	used := map[int]bool{}
	for i, line := range lines {
		number, expression, answer, ok := splitEquation(line.Text)
		if !ok {
			continue
		}
		item := CheckedQuestion{Number: number, Expression: expression, SubmittedAnswer: answer, Page: line.Page, Box: line.Box, Confidence: line.Confidence, Source: []int{i}, Status: "needs_review"}
		if number == 0 && line.Box.Valid() {
			for _, left := range lines {
				if left.Page != line.Page || !sameRow(line.Box, left.Box) || left.Box[2] > line.Box[0] || line.Box[0]-left.Box[2] > 4*(line.Box[3]-line.Box[1]) {
					continue
				}
				if m := standaloneLabel.FindStringSubmatch(NormalizeMath(left.Text)); m != nil {
					item.Number, _ = strconv.Atoi(m[1])
					item.Confidence = combineConfidence(item.Confidence, left.Confidence)
				}
			}
		}
		ambiguous := false
		if answer == "" || answer == "()" || strings.Trim(answer, "_.") == "" {
			candidates := []int{}
			for j, other := range lines {
				if i == j || anchors[j] || used[j] || other.Page != line.Page || !sameRow(line.Box, other.Box) || other.Box[0] < line.Box[2]-0.25*min(line.Box[3]-line.Box[1], other.Box[3]-other.Box[1]) {
					continue
				}
				gap := other.Box[0] - line.Box[2]
				if gap > max(line.Box[3]-line.Box[1], other.Box[3]-other.Box[1])*8 {
					continue
				}
				// Stop at the next equation in this row, preserving worksheet columns.
				blocked := false
				for k := range anchors {
					next := lines[k]
					if k != i && next.Page == line.Page && sameRow(line.Box, next.Box) && next.Box[0] > line.Box[0] && other.Box[0] >= next.Box[0] {
						blocked = true
					}
				}
				if blocked {
					continue
				}
				normalized := NormalizeMath(other.Text)
				if normalized != "" && !strings.ContainsAny(normalized, "√✓✔=<>≤≥") {
					candidates = append(candidates, j)
				}
			}
			if len(candidates) == 1 {
				j := candidates[0]
				used[j] = true
				other := lines[j]
				item.SubmittedAnswer = NormalizeMath(other.Text)
				item.Source = append(item.Source, j)
				item.Confidence = combineConfidence(item.Confidence, other.Confidence)
				item.Box = paddleocr.Box{min(item.Box[0], other.Box[0]), min(item.Box[1], other.Box[1]), max(item.Box[2], other.Box[2]), max(item.Box[3], other.Box[3])}
			} else if len(candidates) > 1 {
				ambiguous = true
			}
		}
		expected, equal, err := comparison(expression, item.SubmittedAnswer)
		item.ExpectedAnswer = expected
		if err == nil && !remainderAnswer.MatchString(NormalizeMath(item.SubmittedAnswer)) {
			calculated, _ := Evaluate(expression)
			submitted, _ := answerValue(item.SubmittedAnswer)
			item.Difference = new(big.Rat).Sub(calculated, submitted).RatString()
		}
		if err != nil {
			item.Reason = err.Error()
		} else {
			if equal {
				item.Comparison = "correct"
			} else {
				item.Comparison = "incorrect"
			}
		}
		switch {
		case ambiguous:
			item.Reason = "multiple possible answers on this row"
		case err != nil: // Keep unparsed or missing answers pending review.
		case item.Confidence != nil && o.MinConfidence > 0 && (math.IsNaN(*item.Confidence) || math.IsInf(*item.Confidence, 0) || *item.Confidence < o.MinConfidence || *item.Confidence > 1):
			item.Reason = "OCR confidence is below the threshold"
		default:
			item.Status = item.Comparison
		}
		report.Questions = append(report.Questions, item)
	}
	// Numbered worksheets read in question order, otherwise in page/row order.
	sort.SliceStable(report.Questions, func(i, j int) bool {
		a, b := report.Questions[i], report.Questions[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		an, bn := a.Number, b.Number
		if an == 0 {
			an = 10000
		}
		if bn == 0 {
			bn = 10000
		}
		if an != bn {
			return an < bn
		}
		if a.Box[1] != b.Box[1] {
			return a.Box[1] < b.Box[1]
		}
		return a.Box[0] < b.Box[0]
	})
	counts := map[string]int{}
	seenNumbers := map[int]bool{}
	maxNumber := 0
	for _, q := range report.Questions {
		if q.Number > 0 {
			counts[fmt.Sprintf("%d/%d", q.Page, q.Number)]++
			seenNumbers[q.Number] = true
			maxNumber = max(maxNumber, q.Number)
		}
	}
	for i := range report.Questions {
		q := &report.Questions[i]
		if q.Number > 0 && counts[fmt.Sprintf("%d/%d", q.Page, q.Number)] > 1 {
			q.Status = "needs_review"
			q.Reason = "duplicate question number"
		}
		if q.Comparison == "correct" {
			report.MatchingValues++
		} else if q.Comparison == "incorrect" {
			report.MismatchingValues++
		}
		switch q.Status {
		case "correct":
			report.Correct++
		case "incorrect":
			report.Incorrect++
		default:
			report.NeedsReview++
		}
	}
	report.Detected = len(report.Questions)
	if report.Detected == 0 {
		report.Warnings = append(report.Warnings, "No supported equations were detected; this is not a passing result.")
	}
	limit := maxNumber
	if o.ExpectedCount > 0 {
		limit = o.ExpectedCount
	}
	limit = min(limit, 1000)
	if len(seenNumbers) > 0 {
		for n := 1; n <= limit; n++ {
			if !seenNumbers[n] {
				report.MissingNumbers = append(report.MissingNumbers, n)
			}
		}
	}
	if o.ExpectedCount > 0 && report.Detected != o.ExpectedCount {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Expected %d questions but detected %d.", o.ExpectedCount, report.Detected))
	}
	if len(report.MissingNumbers) > 0 {
		report.Warnings = append(report.Warnings, "Some question numbers were not recognized.")
	}
	report.Warnings = append(report.Warnings, "Counts cover detected equations only; OCR can miss or misread handwritten content.")
	return report, nil
}
