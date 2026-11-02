package arithmetic

import (
	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
	"testing"
)

func checkEngine(t *testing.T, engine string, texts ...string) CheckReport {
	t.Helper()
	lines := []paddleocr.Line{}
	for _, text := range texts {
		lines = append(lines, paddleocr.Line{Text: text, Page: 1})
	}
	r, err := Check(lines, CheckOptions{ExpectedCount: 4})
	if err != nil {
		t.Fatal(err)
	}
	r.Engine = engine
	return r
}
func TestMergeChecksEitherPassesAndVisionWinsOtherwise(t *testing.T) {
	vision := checkEngine(t, "vision", "(1)12*3=35", "(2)14*2=28", "(3)15*2=29", "(4)13*2=t")
	paddle := checkEngine(t, "paddleocr", "(1)12*3=36", "(2)14*2=27", "(3)15*2=28", "(4)13*2=25")
	r := MergeChecks(vision, paddle)
	if r.Detected != 4 || r.Correct != 2 || r.Incorrect != 1 || r.NeedsReview != 1 {
		t.Fatalf("counts: %+v", r)
	}
	for i, q := range r.Questions {
		if len(q.Readings) != 2 {
			t.Fatalf("missing evidence: %+v", q)
		}
		if i == 0 && (q.OCREngine != "paddleocr" || q.SubmittedAnswer != "36") {
			t.Fatal(q)
		}
		if i == 2 && (q.OCREngine != "vision" || q.SubmittedAnswer != "29") {
			t.Fatal(q)
		}
		for _, reading := range q.Readings {
			for _, n := range reading.Source {
				if n < 0 || n >= len(r.Lines) {
					t.Fatal("bad source index")
				}
			}
		}
	}
}
func TestMergeChecksAlignsUnnumberedAndMalformedPaddleLabels(t *testing.T) {
	vision := checkEngine(t, "vision", "(1)12*3=36", "(2)14*2=28")
	paddle := checkEngine(t, "paddleocr", "12*3=36", `\begin{array}{c}(2)14*2=28`)
	r := MergeChecks(vision, paddle)
	if r.Detected != 2 || r.Correct != 2 || len(r.Questions[1].Readings) != 2 {
		t.Fatalf("%+v", r)
	}
}
func TestMergeChecksKeepsDifferentQuestionIdentities(t *testing.T) {
	a := checkEngine(t, "vision", "(1)12*3=36", "(2)12*3=36")
	b := checkEngine(t, "paddleocr", "(1)12*3=35", "(2)12*3=35")
	r := MergeChecks(a, b)
	if r.Detected != 2 || r.Correct != 2 {
		t.Fatalf("%+v", r)
	}
}

func TestMergeChecksPreservesNestedAlternativeEvidence(t *testing.T) {
	original := checkEngine(t, "vision", "(1)12*3=t")
	alternative := checkEngine(t, "vision:deskew", "(1)12*3=36")
	native := MergeChecks(original, alternative)
	native.Engine = "vision"
	paddle := checkEngine(t, "paddleocr", "(1)12*3=35")
	result := MergeChecks(native, paddle)
	q := result.Questions[0]
	if len(q.Readings) != 3 || q.OCREngine != "vision:deskew" {
		t.Fatal(q)
	}
	for _, reading := range q.Readings {
		if len(reading.Source) != 1 || result.Lines[reading.Source[0]].Text == "" {
			t.Fatal(reading)
		}
	}
	if result.Lines[q.Readings[1].Source[0]].Text != "(1)12*3=36" {
		t.Fatal("alternative source lost")
	}
}
