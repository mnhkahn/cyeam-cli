package reading

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestGradeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		score float64
		grade string
	}{{0, "差"}, {39.99, "差"}, {40, "中"}, {50, "中"}, {60, "中"}, {60.01, "良"}, {79.99, "良"}, {80, "优"}, {100, "优"}, {-1, ""}, {101, ""}, {math.NaN(), ""}} {
		if got := GradeForScore(tc.score); got != tc.grade {
			t.Fatalf("%v: %s != %s", tc.score, got, tc.grade)
		}
	}
}
func TestFocusGroupsRepeatedDirectionAndPreservesWordForms(t *testing.T) {
	r, err := Review("Robotics club loves robots. Robotics club makes robots.", "Robot club loves robot. Robot club makes robot.", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.FocusWords, []string{"robotics", "robots"}) {
		t.Fatal(r.FocusWords, r.Substitutions)
	}
	if r.PracticeURL != "https://www.cyeam.com/ai/translate?words=robotics,robots" {
		t.Fatal(r.PracticeURL)
	}
	if r.Substitutions[0].Count != 2 || r.Substitutions[1].Count != 2 {
		t.Fatal(r.Substitutions)
	}
	if !r.Provisional || !strings.Contains(r.Feedback, r.PracticeURL) {
		t.Fatal(r.Feedback)
	}
}
func TestContractionsSpellingAndHomophonesAreEquivalent(t *testing.T) {
	for _, pair := range [][2]string{
		{"It's brilliant. We're ready. We meet in school.", "Its brilliant. We are ready. We meat in school."},
		{"He's been helpful. I don't worry.", "He has been helpful. I do not worry."},
		{"Our robot is ready.", "Our robot is ready."},
	} {
		r, err := Review(pair[0], pair[1], Options{})
		if err != nil {
			t.Fatal(err)
		}
		if r.Substituted != 0 || r.Score == nil || *r.Score != 100 || r.Grade != "优" || !r.Provisional {
			t.Fatalf("%+v", r)
		}
	}
}
func TestExtraSubjectAndPossessiveAreNotSilentlyErased(t *testing.T) {
	r, err := Review("Our robot is ready.", "Our robot it's ready.", Options{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range r.Differences {
		if d.Kind == "insert" && d.Transcript == "it" {
			found = true
		}
	}
	if !found {
		t.Fatal(r.Differences)
	}
	a, _, _ := normalize("John's robot", "reference", nil)
	if a[0].word != "john's" {
		t.Fatal(a)
	}
}
func TestFunctionWordsAndMissingTextDoNotReduceScore(t *testing.T) {
	r, err := Review("Robot is bright to people, really helpful.", "Robot has bright for people, helpful.", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Score == nil || *r.Score != 100 || r.Substituted != 0 || r.Deleted != 1 || len(r.FocusWords) != 0 {
		t.Fatalf("%+v", r)
	}
	ignored := 0
	for _, d := range r.Differences {
		if d.Ignored && d.Kind == "substitute" {
			ignored++
		}
	}
	if ignored != 2 {
		t.Fatal(r.Differences)
	}
}
func TestRepetitionsAndTooLittleEvidence(t *testing.T) {
	r, err := Review("Robots help people.", "Robots robots help people.", Options{})
	if err != nil || r.Score == nil || *r.Score != 100 {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = Review("They had had lunch.", "They had had lunch.", Options{})
	if err != nil || r.Deleted != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = Review("Robots help people.", "Robots", Options{})
	if err != nil || r.Grade != "" || r.Score != nil {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestHumanGradeOverridesWithoutChangingEvidence(t *testing.T) {
	r, err := Review("Robots help people.", "Robots help people.", Options{ConfirmedGrade: "中"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Grade != "中" || r.Provisional || *r.Score != 100 || r.GradeSource != "human_confirmed" {
		t.Fatalf("%+v", r)
	}
	if !strings.Contains(r.Feedback, "家长听音确认") {
		t.Fatal(r.Feedback)
	}
}
func TestParseReportRecomputesTamperedFeedbackAndGrade(t *testing.T) {
	original, err := Review("Robots help people.", "Robots help people.", Options{ConfirmedGrade: "中"})
	if err != nil {
		t.Fatal(err)
	}
	original.Grade = "100"
	original.Feedback = "invented advice"
	original.PracticeURL = "https://wrong.example"
	body, _ := json.Marshal(original)
	for _, data := range [][]byte{body, append(append([]byte(`{"ok":true,"data":`), body...), '}')} {
		r, err := ParseReport(data)
		if err != nil {
			t.Fatal(err)
		}
		if r.Grade != "中" || r.PracticeURL != "" || strings.Contains(r.Feedback, "invented") {
			t.Fatal(r.Feedback)
		}
	}
	original.Transcript = "changed"
	body, _ = json.Marshal(original)
	if _, err := ParseReport(body); err == nil {
		t.Fatal("accepted wrong fingerprint")
	}
}
func TestInvalidReadingInput(t *testing.T) {
	for _, pair := range [][2]string{{"", "hello"}, {"hello", "😔"}, {strings.Repeat("word ", 2001), "word"}} {
		if _, err := Review(pair[0], pair[1], Options{}); err == nil {
			t.Fatal("accepted invalid input")
		}
	}
	if _, err := Review("robot", "robot", Options{ConfirmedGrade: "A"}); err == nil {
		t.Fatal("accepted grade")
	}
}
