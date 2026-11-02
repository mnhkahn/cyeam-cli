package visionocr

import (
	"context"
	"errors"
	"testing"
)

func TestParseRequiresActualText(t *testing.T) {
	for _, s := range []string{"not-json", `{}`, `{"lines":[]}`} {
		if _, err := Parse([]byte(s)); err == nil {
			t.Fatal(s)
		}
	}
	r, err := Parse([]byte(`{"lines":[{"text":"(1)12*3=t","page":1}],"alternatives":[{"kind":"deskew","line":{"text":"(1)12*3=36","page":1}}]}`))
	if err != nil || len(r.Lines) != 1 || len(r.Alternatives) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestCanceledNativeRequestStops(t *testing.T) {
	if !Available() {
		t.Skip("Apple Vision not available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RecognizeRaw(ctx, []byte("unused")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
