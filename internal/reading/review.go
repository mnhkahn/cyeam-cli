package reading

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/wordstudy"
)

const SchemaVersion = 1
const maxWords = 2000

type Options struct {
	ConfirmedGrade string `json:"confirmed_grade,omitempty"`
}
type Difference struct {
	Kind              string `json:"kind"`
	Reference         string `json:"reference,omitempty"`
	Transcript        string `json:"transcript,omitempty"`
	ReferencePosition int    `json:"reference_position"`
	Ignored           bool   `json:"ignored,omitempty"`
}
type Substitution struct {
	Reference     string `json:"reference"`
	Transcript    string `json:"transcript"`
	Count         int    `json:"count"`
	FirstPosition int    `json:"first_position"`
}
type Report struct {
	SchemaVersion     int             `json:"schema_version"`
	Reference         string          `json:"reference"`
	Transcript        string          `json:"transcript"`
	ReferenceSHA256   string          `json:"reference_sha256"`
	TranscriptSHA256  string          `json:"transcript_sha256"`
	Options           Options         `json:"options"`
	ReferenceWords    int             `json:"reference_words"`
	TranscriptWords   int             `json:"transcript_words"`
	Matched           int             `json:"matched"`
	Substituted       int             `json:"substituted"`
	Deleted           int             `json:"deleted"`
	Inserted          int             `json:"inserted"`
	Comparable        int             `json:"comparable"`
	EligibleReference int             `json:"eligible_reference"`
	Score             *float64        `json:"reference_score,omitempty"`
	Grade             string          `json:"grade,omitempty"`
	Provisional       bool            `json:"provisional"`
	GradeSource       string          `json:"grade_source"`
	Substitutions     []Substitution  `json:"substitutions"`
	FocusWords        []string        `json:"focus_words"`
	PracticeURL       string          `json:"practice_url,omitempty"`
	Normalizations    []Normalization `json:"normalizations"`
	Differences       []Difference    `json:"differences"`
	Warnings          []string        `json:"warnings"`
	Feedback          string          `json:"feedback"`
}

func validGrade(s string) bool {
	switch s {
	case "", "100", "优", "良", "中", "差":
		return true
	}
	return false
}
func digest(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }

func Review(reference, transcript string, opts Options) (Report, error) {
	r := Report{SchemaVersion: SchemaVersion, Reference: reference, Transcript: transcript, Options: opts, ReferenceSHA256: digest(reference), TranscriptSHA256: digest(transcript), Substitutions: []Substitution{}, FocusWords: []string{}, Differences: []Difference{}}
	if !validGrade(opts.ConfirmedGrade) {
		return r, fmt.Errorf("confirmed grade must be 100, 优, 良, 中 or 差")
	}
	a, ea, na := normalize(reference, "reference", nil)
	b, eb, nb := normalize(transcript, "transcript", referenceRepeats(reference))
	r.ReferenceWords = na
	r.TranscriptWords = nb
	r.Normalizations = append(ea, eb...)
	if len(a) == 0 || len(b) == 0 {
		return r, fmt.Errorf("reference and transcript must each contain English words")
	}
	if len(a) > maxWords || len(b) > maxWords {
		return r, fmt.Errorf("reading comparison supports at most %d normalized words per input", maxWords)
	}
	for _, t := range a {
		if !functionWords[t.word] {
			r.EligibleReference++
		}
	}
	r.Differences = align(a, b)
	counts := map[string]*Substitution{}
	for _, d := range r.Differences {
		if d.Ignored {
			continue
		}
		switch d.Kind {
		case "match", "equivalent":
			r.Matched++
		case "substitute":
			r.Substituted++
			key := d.Reference + "\x00" + d.Transcript
			if counts[key] == nil {
				counts[key] = &Substitution{Reference: d.Reference, Transcript: d.Transcript, FirstPosition: d.ReferencePosition}
			}
			counts[key].Count++
		case "delete":
			r.Deleted++
		case "insert":
			r.Inserted++
		}
	}
	for _, s := range counts {
		r.Substitutions = append(r.Substitutions, *s)
	}
	sort.Slice(r.Substitutions, func(i, j int) bool {
		a, b := r.Substitutions[i], r.Substitutions[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.FirstPosition != b.FirstPosition {
			return a.FirstPosition < b.FirstPosition
		}
		if a.Reference != b.Reference {
			return a.Reference < b.Reference
		}
		return a.Transcript < b.Transcript
	})
	seen := map[string]bool{}
	for _, s := range r.Substitutions {
		if s.Count >= 2 && !seen[s.Reference] {
			r.FocusWords = append(r.FocusWords, s.Reference)
			seen[s.Reference] = true
		}
	}
	r.PracticeURL = wordstudy.PracticeURL(r.FocusWords)
	r.Comparable = r.Matched + r.Substituted
	// Missing text is not scored as wrong. Require enough aligned material to
	// avoid awarding a high grade to a lone word in a long passage.
	scorable := r.Comparable >= min(3, r.EligibleReference) && r.Comparable > 0 && r.Comparable*4 >= r.EligibleReference
	if scorable {
		score := 100 * float64(r.Matched) / float64(r.Comparable)
		r.Score = &score
		r.Grade = GradeForScore(score)
		r.Provisional = true
		r.GradeSource = "asr_reference"
	} else {
		r.GradeSource = "insufficient_evidence"
		r.Warnings = append(r.Warnings, "Too few comparable words; no automatic grade was assigned.")
	}
	if opts.ConfirmedGrade != "" {
		r.Grade = opts.ConfirmedGrade
		r.Provisional = false
		r.GradeSource = "human_confirmed"
	}
	r.Warnings = append(r.Warnings, "ASR substitutions are review clues, not confirmed pronunciation errors. Missing/extra words and configured function words do not affect the reference score.")
	r.Feedback = Feedback(r)
	return r, nil
}
func GradeForScore(score float64) string {
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 100 {
		return ""
	}
	switch {
	case score >= 80:
		return "优"
	case score > 60:
		return "良"
	case score >= 40:
		return "中"
	default:
		return "差"
	}
}
func Feedback(r Report) string {
	var text strings.Builder
	switch {
	case r.Grade == "":
		text.WriteString("阅读等级：待核对（可比较材料不足）。\n")
	case r.Provisional:
		fmt.Fprintf(&text, "阅读等级：暂评·%s（ASR辅助评价）。\n", r.Grade)
	default:
		fmt.Fprintf(&text, "阅读等级：%s（家长听音确认）。\n", r.Grade)
	}
	if r.Score != nil {
		fmt.Fprintf(&text, "阅读参考分：%.1f%%（可比较词%d次，仅计算匹配与替换，过滤功能词，不含缺失和多出）。\n", *r.Score, r.Comparable)
	}
	if r.PracticeURL != "" {
		fmt.Fprintf(&text, "\n重点词重读：%s\n\n打开链接，逐词听发音并跟读3遍，注意单词读完整、读清词尾。再回到原文，把含这些词的句子完整读一遍，最后重录相关段落。\n", r.PracticeURL)
	} else {
		text.WriteString("\n本次未发现同方向重复至少2次的实词替换。请对照原文听音复读；转写不清的片段需核听。\n")
	}
	text.WriteString("\n转写差异仅供复读参考，不直接认定为朗读错误。\n")
	return text.String()
}

// ValidateReference rejects unusable passages before an audio upload starts.
func ValidateReference(reference string) error {
	words, _, _ := normalize(reference, "reference", nil)
	if len(words) == 0 {
		return fmt.Errorf("reference must contain English words")
	}
	if len(words) > maxWords {
		return fmt.Errorf("reading comparison supports at most %d normalized words per input", maxWords)
	}
	return nil
}
