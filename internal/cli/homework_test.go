package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/homework"
)

type fakeHomeworkService struct {
	review         func(context.Context, homework.ReviewRequest) (json.RawMessage, error)
	regrade        func(context.Context, json.RawMessage) (json.RawMessage, error)
	reading        func(context.Context, string, string, []byte) (json.RawMessage, error)
	readingRegrade func(context.Context, json.RawMessage) (json.RawMessage, error)
}

func (f *fakeHomeworkService) Review(ctx context.Context, input homework.ReviewRequest) (json.RawMessage, error) {
	if f.review == nil {
		return nil, fmt.Errorf("unexpected Review call")
	}
	return f.review(ctx, input)
}
func (f *fakeHomeworkService) Regrade(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	if f.regrade == nil {
		return nil, fmt.Errorf("unexpected Regrade call")
	}
	return f.regrade(ctx, input)
}
func (f *fakeHomeworkService) Reading(ctx context.Context, reference, filename string, audio []byte) (json.RawMessage, error) {
	if f.reading == nil {
		return nil, fmt.Errorf("unexpected Reading call")
	}
	return f.reading(ctx, reference, filename, audio)
}
func (f *fakeHomeworkService) ReadingRegrade(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	if f.readingRegrade == nil {
		return nil, fmt.Errorf("unexpected ReadingRegrade call")
	}
	return f.readingRegrade(ctx, input)
}

func TestHomeworkInputFromJSONPreservesInput(t *testing.T) {
	input, err := homeworkInputFromJSON([]byte(`{"ok":true,"data":{"input":{"kind":"arithmetic","count":1,"items":[],"future":"kept"},"report":{"grade":"100"}}}`), "arithmetic")
	if err != nil {
		t.Fatal(err)
	}
	if string(input) != `{"kind":"arithmetic","count":1,"items":[],"future":"kept"}` {
		t.Fatal(string(input))
	}
	if _, err := homeworkInputFromJSON([]byte(`{"input":{"kind":"dictation"}}`), "arithmetic"); err == nil {
		t.Fatal("accepted the wrong homework kind")
	}
}
