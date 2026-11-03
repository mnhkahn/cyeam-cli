// Package wordstudy shares the vocabulary practice entry point across homework
// creation and reading feedback.
package wordstudy

import (
	"net/url"
	"strings"
)

func PracticeURL(words []string) string {
	parts := []string{}
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" {
			parts = append(parts, strings.ReplaceAll(url.QueryEscape(word), "+", "%20"))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "https://www.cyeam.com/ai/translate?words=" + strings.Join(parts, ",")
}
