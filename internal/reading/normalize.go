// Package reading deterministically compares a supplied passage and ASR text.
// Its scores describe text alignment, never verified pronunciation accuracy.
package reading

import (
	"regexp"
	"strings"
)

var wordPattern = regexp.MustCompile(`[a-z]+(?:'[a-z]+)*`)
var functionWords = wordSet("am is are was were be been being a an the to of in on at for from with by and or but")
var perfectFollowing = wordSet("been done gone seen had got")
var contractions = map[string][]string{
	"i'm": {"i", "am"}, "you're": {"you", "are"}, "we're": {"we", "are"}, "they're": {"they", "are"},
	"i've": {"i", "have"}, "you've": {"you", "have"}, "we've": {"we", "have"}, "they've": {"they", "have"},
	"i'll": {"i", "will"}, "you'll": {"you", "will"}, "he'll": {"he", "will"}, "she'll": {"she", "will"}, "we'll": {"we", "will"}, "they'll": {"they", "will"}, "it'll": {"it", "will"},
	"can't": {"can", "not"}, "cannot": {"can", "not"}, "won't": {"will", "not"}, "shan't": {"shall", "not"},
	"isn't": {"is", "not"}, "aren't": {"are", "not"}, "wasn't": {"was", "not"}, "weren't": {"were", "not"},
	"don't": {"do", "not"}, "doesn't": {"does", "not"}, "didn't": {"did", "not"}, "haven't": {"have", "not"}, "hasn't": {"has", "not"}, "hadn't": {"had", "not"},
	"couldn't": {"could", "not"}, "wouldn't": {"would", "not"}, "shouldn't": {"should", "not"}, "mustn't": {"must", "not"},
}
var pronounS = wordSet("it's he's she's that's there's what's who's where's how's")

type token struct {
	word     string
	position int
}
type Normalization struct {
	Source     string `json:"source"`
	Kind       string `json:"kind"`
	Original   string `json:"original"`
	Normalized string `json:"normalized"`
}

func wordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(s) {
		out[w] = true
	}
	return out
}
func normalize(text, source string, protectedRepeats map[string]bool) ([]token, []Normalization, int) {
	lower := strings.ToLower(strings.NewReplacer("’", "'", "‘", "'").Replace(text))
	spans := wordPattern.FindAllStringIndex(lower, -1)
	raw := make([]string, len(spans))
	for i, span := range spans {
		raw[i] = lower[span[0]:span[1]]
	}
	out := []token{}
	events := []Normalization{}
	for i, w := range raw {
		if source == "transcript" && i > 0 && raw[i-1] == w && !protectedRepeats[w] && strings.TrimSpace(lower[spans[i-1][1]:spans[i][0]]) == "" {
			events = append(events, Normalization{source, "adjacent_repeat", w + " " + w, w})
			continue
		}
		parts := contractions[w]
		if pronounS[w] {
			auxiliary := "is"
			// Restrict the ambiguous 's expansion to pronouns and known perfect
			// contexts. Possessives and 'd are left intact rather than guessed.
			next := i + 1
			for next < len(raw) && (raw[next] == "already" || raw[next] == "just" || raw[next] == "recently") {
				next++
			}
			if next < len(raw) && perfectFollowing[raw[next]] {
				auxiliary = "has"
			}
			parts = []string{strings.TrimSuffix(w, "'s"), auxiliary}
		}
		if len(parts) == 0 {
			parts = []string{w}
		} else {
			events = append(events, Normalization{source, "contraction", w, strings.Join(parts, " ")})
		}
		for _, part := range parts {
			out = append(out, token{part, i})
		}
	}
	return out, events, len(raw)
}
func equivalentWord(a, b string) bool {
	return a == b || (a == "meet" && b == "meat") || (a == "meat" && b == "meet")
}

func referenceRepeats(text string) map[string]bool {
	raw := wordPattern.FindAllString(strings.ToLower(strings.NewReplacer("’", "'", "‘", "'").Replace(text)), -1)
	out := map[string]bool{}
	for i := 1; i < len(raw); i++ {
		if raw[i] == raw[i-1] {
			out[raw[i]] = true
		}
	}
	return out
}
