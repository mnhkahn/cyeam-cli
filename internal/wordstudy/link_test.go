package wordstudy

import (
	"net/url"
	"testing"
)

func TestPracticeURLKeepsCommaAndEscapesEntries(t *testing.T) {
	link := PracticeURL([]string{"robotics", "robots"})
	if link != "https://www.cyeam.com/ai/translate?words=robotics,robots" {
		t.Fatal(link)
	}
	link = PracticeURL([]string{"post office", "word&other=true"})
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if len(u.Query()) != 1 || u.Query().Get("words") != "post office,word&other=true" {
		t.Fatal(link)
	}
	if PracticeURL(nil) != "" {
		t.Fatal("empty link")
	}
}
