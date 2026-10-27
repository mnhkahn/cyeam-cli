package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestReviewGradePreservesAssignmentAndReplacesOnlyHeader(t *testing.T) {
	for _, description := range []string{"", "作业类型：口算\n作业内容：\n12 + 9 = ___\n", "打分结果：差\n作业类型：口算\n作业内容：\n12 + 9 = ___\n", "打分结果:差"} {
		result := withReviewGrade(description, "优", false)
		if !strings.HasPrefix(result, "打分结果：优") {
			t.Fatal(result)
		}
		if strings.Contains(description, "作业类型") && !strings.HasSuffix(result, "作业类型：口算\n作业内容：\n12 + 9 = ___\n") {
			t.Fatal("assignment changed:", result)
		}
		if got := withReviewGrade(result, "良", true); strings.Count(got, "打分结果") != 1 || !strings.HasPrefix(got, "打分结果：暂评·良") {
			t.Fatal(got)
		}
	}
	// A similarly named line inside the question payload is not metadata.
	description := "作业内容：\n打分结果：请填写评分理由"
	if got := withReviewGrade(description, "良", false); !strings.HasSuffix(got, description) {
		t.Fatal("question changed", got)
	}
}

func TestReviewGradeRetryDoesNotDuplicateComment(t *testing.T) {
	opts := reviewOpts(t, "text")
	opts.Grade = "优"
	opts.Provisional = true
	posts, puts := 0, 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /cards/card-1/actions":
			fmt.Fprint(w, `[]`)
		case "POST /cards/card-1/actions/comments":
			posts++
			r.ParseForm()
			var payload reviewPayload
			if err := json.Unmarshal([]byte(strings.TrimPrefix(r.FormValue("text"), ReviewPrefix)), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Grade != "优" || !payload.Provisional {
				t.Errorf("missing grade in history: %+v", payload)
			}
			fmt.Fprint(w, `{"id":"action"}`)
		case "GET /cards/card-1":
			fmt.Fprint(w, `{"desc":"作业内容：计算第3题"}`)
		case "PUT /cards/card-1":
			puts++
			r.ParseForm()
			if r.FormValue("desc") != "打分结果：暂评·优\n作业内容：计算第3题" {
				t.Errorf("wrong grade description: %q", r.FormValue("desc"))
			}

			if puts == 1 {
				http.Error(w, "temporary error", 500)
				return
			}
			fmt.Fprint(w, `{}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	if _, err := client.PublishReview(context.Background(), opts); err == nil {
		t.Fatal("expected grade write failure")
	}
	result, err := client.PublishReview(context.Background(), opts)
	if err != nil || result.Grade != "优" || !result.Provisional {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := client.PublishReview(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || puts != 2 {
		t.Fatalf("posts=%d puts=%d", posts, puts)
	}
	opts.Grade = "良"
	if _, err := client.PublishReview(context.Background(), opts); err == nil {
		t.Fatal("must reject changed grade for same review ID")
	}
}

func TestInvalidReviewGradeNeverUsesNetwork(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("unexpected network call") })
	for _, grade := range []string{"100", "优良", " 暂评"} {
		opts := reviewOpts(t, "text")
		opts.Grade = grade
		if _, err := client.PublishReview(context.Background(), opts); err == nil {
			t.Fatal("accepted invalid grade")
		}
	}
	opts := reviewOpts(t, "text")
	opts.Provisional = true
	if _, err := client.PublishReview(context.Background(), opts); err == nil {
		t.Fatal("provisional needs grade")
	}
}

func TestGradeUnknownWriteIsReconciledWithoutRewriting(t *testing.T) {
	opts := reviewOpts(t, "text")
	state := reviewState{ReviewResult: ReviewResult{CardID: opts.CardID, StatePath: opts.StatePath, Grade: "良"}}
	description := "原题\n作业内容：第1题"
	puts := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]string{"desc": description})
			return
		}
		puts++
		r.ParseForm()
		description = r.FormValue("desc")
		http.Error(w, "response lost after update", 500)
	})
	if err := client.applyReviewGrade(context.Background(), &state); err == nil {
		t.Fatal("expected unknown write response")
	}
	if err := client.applyReviewGrade(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	if puts != 1 || !state.GradeApplied {
		t.Fatalf("puts=%d applied=%v", puts, state.GradeApplied)
	}
}

func TestGradeNeverOverwritesMissingDescription(t *testing.T) {
	opts := reviewOpts(t, "text")
	state := reviewState{ReviewResult: ReviewResult{CardID: opts.CardID, StatePath: opts.StatePath, Grade: "良"}}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatal("must not write when description response is invalid")
		}
		fmt.Fprint(w, `{"id":"card-1"}`)
	})
	if err := client.applyReviewGrade(context.Background(), &state); err == nil {
		t.Fatal("must reject missing description")
	}
}
