package arithmetic

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

func confidence(v float64) *float64 { return &v }
func ocrLine(text string, x, y, w, h float64, score *float64) paddleocr.Line {
	return paddleocr.Line{Text: text, Box: paddleocr.Box{x, y, x + w, y + h}, Page: 1, Confidence: score}
}
func TestEvaluateExactAndAllowlisted(t *testing.T) {
	for _, tc := range []struct{ expression, want string }{
		{"422×11", "4642"}, {"503×28", "14084"}, {"0.1＋0.2", "3/10"}, {"(40+4)×25", "1100"}, {"25÷8", "25/8"}, {"-3*(2-5)", "9"}, {"１２÷３", "4"}, {`$ 422 \\times 11 $`, "4642"}, {`\frac{1}{2}+\frac{1}{4}`, "3/4"},
	} {
		v, err := Evaluate(tc.expression)
		if err != nil || v.RatString() != tc.want {
			t.Fatalf("%s: %v %v", tc.expression, v, err)
		}
	}
	for _, s := range []string{"", "1/0", "1+", "(1+2", "1**2", "1e3", "1; rm -rf /", "os.Exit(0)", "1O+2", "1+2=3", strings.Repeat("(", 33) + "1" + strings.Repeat(")", 33), strings.Repeat("9", 129)} {
		if _, err := Evaluate(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}
func TestCheckPairsAnswersWithoutUsingScratchwork(t *testing.T) {
	lines := []paddleocr.Line{
		ocrLine("(1) 422×11=", 10, 10, 100, 20, confidence(.99)),
		ocrLine("4642", 115, 9, 40, 22, confidence(.97)),
		ocrLine("(2) 503×28=14034", 300, 10, 170, 20, confidence(.95)),
		ocrLine("(3) 166×85=", 10, 100, 100, 20, confidence(.99)),
		ocrLine("14110", 30, 145, 50, 20, confidence(.99)), // correct number, but only in scratch work
		ocrLine("(4) 25÷8=3……1", 300, 100, 170, 20, confidence(.99)),
		ocrLine("(5) 0.1+0.2=0.3", 10, 200, 170, 20, confidence(.99)),
	}
	r, err := Check(lines, CheckOptions{MinConfidence: .9, ExpectedCount: 5})
	if err != nil {
		t.Fatal(err)
	}
	if r.Detected != 5 || r.Correct != 3 || r.Incorrect != 1 || r.NeedsReview != 1 {
		t.Fatalf("%+v", r)
	}
	if r.Questions[0].SubmittedAnswer != "4642" || *r.Questions[0].Confidence != .97 {
		t.Fatal(r.Questions[0])
	}
	if r.Questions[0].Difference != "0" || r.Questions[1].Difference != "50" {
		t.Fatal("incorrect difference")
	}
	if r.Questions[2].SubmittedAnswer != "" || r.Questions[2].Status != "needs_review" {
		t.Fatal("scratch answer was used")
	}
	if r.Questions[3].ExpectedAnswer != "3……1" {
		t.Fatal(r.Questions[3])
	}
}
func TestCheckUncertaintyAmbiguityAndOmissions(t *testing.T) {
	cases := []struct {
		name   string
		lines  []paddleocr.Line
		reason string
	}{
		{"no score", []paddleocr.Line{ocrLine("(1)2+2=4", 0, 0, 100, 20, nil)}, "confidence"},
		{"low score", []paddleocr.Line{ocrLine("(1)2+2=4", 0, 0, 100, 20, confidence(.8))}, "threshold"},
		{"no box", []paddleocr.Line{{Text: "(1)2+2=4", Page: 1, Confidence: confidence(.99)}}, "coordinates"},
		{"garbled answer", []paddleocr.Line{ocrLine("(1)2+2=", 0, 0, 100, 20, confidence(.99)), ocrLine(`$q_{240}$`, 103, 0, 40, 20, confidence(.99))}, "unambiguous"},
		{"ambiguous", []paddleocr.Line{ocrLine("(1)2+2=", 0, 0, 100, 20, confidence(.99)), ocrLine("4", 103, 0, 10, 20, confidence(.99)), ocrLine("7", 116, 0, 10, 20, confidence(.99))}, "multiple"},
		{"unsafe expression", []paddleocr.Line{ocrLine("(1)2+foo()=4", 0, 0, 100, 20, confidence(.99))}, "invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Check(tc.lines, CheckOptions{MinConfidence: .9, ExpectedCount: 3})
			if err != nil {
				t.Fatal(err)
			}
			if r.NeedsReview != 1 || r.Correct+r.Incorrect != 0 || !strings.Contains(r.Questions[0].Reason, tc.reason) {
				t.Fatalf("%+v", r)
			}
			if len(r.MissingNumbers) != 2 {
				t.Fatal(r.MissingNumbers)
			}
		})
	}
	for _, bad := range []CheckOptions{{MinConfidence: math.NaN()}, {MinConfidence: 1.1}, {ExpectedCount: -1}, {ExpectedCount: 1001}} {
		if _, err := Check(nil, bad); err == nil {
			t.Fatal(bad)
		}
	}
	r, err := Check([]paddleocr.Line{{Text: "计算练习"}}, CheckOptions{})
	if err != nil || r.Detected != 0 || len(r.Warnings) == 0 {
		t.Fatal(r, err)
	}
}
func TestCheckDuplicateNumbersAndColumnBoundary(t *testing.T) {
	lines := []paddleocr.Line{ocrLine("(1)2+2=", 0, 0, 100, 20, confidence(.99)), ocrLine("(2)3+3=", 120, 0, 100, 20, confidence(.99)), ocrLine("6", 225, 0, 20, 20, confidence(.99)), ocrLine("(2)1+1=2", 120, 60, 100, 20, confidence(.99))}
	r, err := Check(lines, CheckOptions{MinConfidence: .9})
	if err != nil {
		t.Fatal(err)
	}
	if r.Questions[0].SubmittedAnswer != "" || r.NeedsReview != 3 {
		t.Fatal(r)
	}
}
func TestCheckRealSpottingReplay(t *testing.T) {
	// Recorded OCR evidence only: no photo, credentials, service URL, or image URLs.
	// Recognized handwriting is intentionally not treated as ground truth.
	data, err := os.ReadFile("testdata/paddle_spotting.json")
	if err != nil {
		t.Fatal(err)
	}
	lines, err := paddleocr.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Check(lines, CheckOptions{MinConfidence: .9, ExpectedCount: 18})
	if err != nil {
		t.Fatal(err)
	}
	if r.Detected != 18 || r.NeedsReview != 18 || r.Correct+r.Incorrect != 0 || len(r.MissingNumbers) > 0 {
		t.Fatalf("%+v", r)
	}
	if r.MatchingValues != 10 || r.MismatchingValues != 6 {
		t.Fatal(r.MatchingValues, r.MismatchingValues)
	}
	for i, q := range r.Questions {
		if q.Number != i+1 {
			t.Fatal(q)
		}
	}
	if r.Questions[5].SubmittedAnswer != "" {
		t.Fatal("used question 6 scratch work")
	}
	if r.Questions[13].SubmittedAnswer != "q_{240}" || r.Questions[16].ExpectedAnswer != "6480" {
		t.Fatal(r.Questions[13], r.Questions[16])
	}
}
