package pdf

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestArithmeticPagination(t *testing.T) {
	for _, tc := range []struct {
		name     string
		capacity int
	}{
		{"standard", 100}, {"grade4", 100}, {"large-number", 75}, {"negative", 75}, {"quantity", 50}, {"vertical", 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, extra := range []int{0, 1} {
				data, err := RenderArithmetic([]byte(strings.Repeat("12 + 3 =\r\n\n", tc.capacity+extra)), ArithmeticOptions{Layout: tc.name, Title: "Practice"})
				if err != nil {
					t.Fatal(err)
				}
				pages := regexp.MustCompile(`/Type /Page\b`).FindAll(data, -1)
				if len(pages) != 1+extra {
					t.Fatalf("got %d pages, want %d", len(pages), 1+extra)
				}
				if !bytes.HasPrefix(data, []byte("%PDF-")) {
					t.Fatal("not a PDF")
				}
			}
		})
	}
}

func TestArithmeticInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		src  string
		opts ArithmeticOptions
	}{
		{" \r\n", ArithmeticOptions{}},
		{"1+1=", ArithmeticOptions{Layout: "unknown"}},
		{"1+1=", ArithmeticOptions{Title: "a\nb"}},
		{strings.Repeat("long question ", 100), ArithmeticOptions{Title: "Practice"}},
	} {
		if _, err := RenderArithmetic([]byte(tc.src), tc.opts); err == nil {
			t.Fatalf("expected error for %+v", tc)
		}
	}
}
