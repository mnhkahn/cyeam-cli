package arithmetic

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var embeddedQuestionNumber = regexp.MustCompile(`^[^0-9]*\(([0-9]{1,4})\)`)

// MergeChecks uses reports in preference order (Vision first). A passing
// recognition wins; otherwise the preferred engine's original reading wins.
// Every reading remains attached for audit, including disagreeing stems.
func MergeChecks(reports ...CheckReport) CheckReport {
	out := CheckReport{Questions: []CheckedQuestion{}}
	if len(reports) == 0 {
		return out
	}
	out.ExpectedCount = reports[0].ExpectedCount
	engines := []string{}
	for _, report := range reports {
		offset := len(out.Lines)
		out.Lines = append(out.Lines, report.Lines...)
		if !containsString(engines, report.Engine) {
			engines = append(engines, report.Engine)
		}
		used := map[int]bool{}
		for _, original := range report.Questions {
			q := original
			q.Source = append([]int(nil), q.Source...)
			for i := range q.Source {
				q.Source[i] += offset
			}
			if q.Number == 0 {
				if m := embeddedQuestionNumber.FindStringSubmatch(q.Expression); m != nil {
					q.Number, _ = strconv.Atoi(m[1])
				}
			}
			if q.OCREngine == "" {
				q.OCREngine = report.Engine
			}
			if len(q.Readings) == 0 {
				q.Readings = []OCRReading{{Engine: report.Engine, Expression: q.Expression, SubmittedAnswer: q.SubmittedAnswer, Status: q.Status, Source: q.Source}}
			} else {
				readings := make([]OCRReading, len(q.Readings))
				for i, reading := range q.Readings {
					reading.Source = append([]int(nil), reading.Source...)
					for j := range reading.Source {
						reading.Source[j] += offset
					}
					readings[i] = reading
				}
				q.Readings = readings
			}
			target := -1
			// Prefer explicit question identity. Only fall back to an expression when
			// one side lacks a number, and the match is unique within this page.
			for i, existing := range out.Questions {
				if used[i] || existing.Page != q.Page {
					continue
				}
				match := q.Number > 0 && existing.Number == q.Number
				if !match && (q.Number == 0 || existing.Number == 0) {
					match = NormalizeMath(q.Expression) == NormalizeMath(existing.Expression)
				}
				if match {
					if target >= 0 {
						target = -2
						break
					}
					target = i
				}
			}
			if target < 0 {
				out.Questions = append(out.Questions, q)
				used[len(out.Questions)-1] = true
				continue
			}
			used[target] = true
			current := out.Questions[target]
			readings := append(append([]OCRReading(nil), current.Readings...), q.Readings...)
			if current.Status != "correct" && q.Status == "correct" {
				if q.Number == 0 {
					q.Number = current.Number
				}
				current = q
			}
			current.Readings = readings
			out.Questions[target] = current
		}
	}
	out.Engine = strings.Join(engines, "+")
	sort.SliceStable(out.Questions, func(i, j int) bool {
		a, b := out.Questions[i], out.Questions[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if a.Number != b.Number {
			if a.Number == 0 {
				return false
			}
			if b.Number == 0 {
				return true
			}
			return a.Number < b.Number
		}
		if a.Box[1] != b.Box[1] {
			return a.Box[1] < b.Box[1]
		}
		return a.Box[0] < b.Box[0]
	})
	seen := map[int]bool{}
	limit := 0
	for _, q := range out.Questions {
		if q.Number > 0 {
			seen[q.Number] = true
			limit = max(limit, q.Number)
		}
		switch q.Status {
		case "correct":
			out.Correct++
		case "incorrect":
			out.Incorrect++
		default:
			out.NeedsReview++
		}
		switch q.Comparison {
		case "correct":
			out.MatchingValues++
		case "incorrect":
			out.MismatchingValues++
		}
	}
	out.Detected = len(out.Questions)
	if out.ExpectedCount > 0 {
		limit = out.ExpectedCount
	}
	if len(seen) > 0 {
		for n := 1; n <= min(limit, 1000); n++ {
			if !seen[n] {
				out.MissingNumbers = append(out.MissingNumbers, n)
			}
		}
	}
	if out.Detected == 0 {
		out.Warnings = append(out.Warnings, "No supported equations were detected; this is not a passing result.")
	}
	if out.ExpectedCount > 0 && out.Detected != out.ExpectedCount {
		out.Warnings = append(out.Warnings, fmt.Sprintf("Expected %d questions but detected %d.", out.ExpectedCount, out.Detected))
	}
	if len(out.MissingNumbers) > 0 {
		out.Warnings = append(out.Warnings, "Some question numbers were not recognized.")
	}
	out.Warnings = append(out.Warnings, "Counts cover detected equations only; OCR can miss or misread handwritten content.")
	return out
}
func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
