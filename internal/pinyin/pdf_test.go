package pinyin

import (
	"regexp"
	"strings"
	"testing"
)

func TestSheetWords(t *testing.T) {
	words := sheetWords("你好，世界\n学习\t中文 " + strings.Repeat("字", 18))
	if strings.Join(words, "|") != "你好|世界|学习|中文|"+strings.Repeat("字", 17)+"|字" {
		t.Fatalf("unexpected blocks: %v", words)
	}
}

func TestSheetPagination(t *testing.T) {
	for _, tc := range []struct{ count, pages int }{{2, 1}, {204, 1}, {205, 2}, {408, 2}, {409, 3}} {
		data, err := GenerateSheetPDF(strings.Repeat("你", tc.count))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(regexp.MustCompile(`/Type /Page\b`).FindAll(data, -1)); n != tc.pages {
			t.Fatalf("%d chars: %d pages, want %d", tc.count, n, tc.pages)
		}
	}
	if _, err := GenerateSheetPDF("abc 123 !"); err == nil {
		t.Fatal("expected error for input without Chinese")
	}
}
