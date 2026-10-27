package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reviewOpts(t *testing.T, kind string) ReviewOptions {
	t.Helper()
	dir := t.TempDir()
	opts := ReviewOptions{CardID: "card-1", ReviewID: "test-review", Type: kind, StatePath: filepath.Join(dir, "review.json")}
	if kind == "text" {
		opts.Text = "第3题：注意\"进位\"。\n其余正确。"
	} else {
		ext := ".png"
		if kind == "audio" {
			ext = ".m4a"
		}
		opts.File = filepath.Join(dir, "input"+ext)
		if err := os.WriteFile(opts.File, []byte("review bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return opts
}

func TestPublishReviewProtocolAndRetry(t *testing.T) {
	for _, kind := range []string{"text", "image", "audio"} {
		for _, lost := range []string{"none", "upload", "comment"} {
			if kind == "text" && lost == "upload" {
				continue
			}
			t.Run(kind+"/"+lost, func(t *testing.T) {
				opts := reviewOpts(t, kind)
				var attachments []reviewAttachment
				var comments []reviewComment
				uploads, posts := 0, 0
				client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
					switch {
					case r.Method == "GET" && r.URL.Path == "/cards/card-1/attachments":
						json.NewEncoder(w).Encode(attachments)
					case r.Method == "POST" && r.URL.Path == "/cards/card-1/attachments":
						uploads++
						if err := r.ParseMultipartForm(1 << 20); err != nil {
							t.Error(err)
							return
						}
						f, h, err := r.FormFile("file")
						if err != nil {
							t.Error(err)
							return
						}
						defer f.Close()
						body, _ := io.ReadAll(f)
						mimeType, _ := ReviewFileType(kind, opts.File)
						name := "homework-review-" + kind + "-test-review" + filepath.Ext(opts.File)
						if string(body) != "review bytes" || r.FormValue("name") != name || h.Filename != name || h.Header.Get("Content-Type") != mimeType {
							t.Errorf("invalid upload: %s, %v", body, h)
						}
						att := reviewAttachment{ID: "att-1", Name: name, MIMEType: mimeType, IsUpload: true}
						attachments = append(attachments, att)
						if lost == "upload" {
							fmt.Fprint(w, "truncated")
							return
						}
						json.NewEncoder(w).Encode(att)
					case r.Method == "GET" && r.URL.Path == "/cards/card-1/actions":
						if r.URL.Query().Get("filter") != "commentCard" {
							t.Error("must only fetch comments")
						}
						json.NewEncoder(w).Encode(comments)
					case r.Method == "POST" && r.URL.Path == "/cards/card-1/actions/comments":
						posts++
						if err := r.ParseForm(); err != nil {
							t.Error(err)
							return
						}
						text := r.PostForm.Get("text")
						if !strings.HasPrefix(text, ReviewPrefix) {
							t.Error(text)
						}
						var payload map[string]any
						if err := json.Unmarshal([]byte(strings.TrimPrefix(text, ReviewPrefix)), &payload); err != nil {
							t.Error(err)
						}
						if len(payload) != 3 || payload["version"] != float64(1) || payload["type"] != kind {
							t.Errorf("payload: %#v", payload)
						}
						if kind == "text" {
							if payload["text"] != opts.Text {
								t.Error(payload)
							}
						} else {
							if len(attachments) != 1 || payload["attachment_id"] != "att-1" {
								t.Error("must upload first", payload)
							}
						}
						comment := reviewComment{ID: "action-1"}
						comment.Data.Text = text
						comments = append(comments, comment)
						if lost == "comment" {
							fmt.Fprint(w, "truncated")
							return
						}
						fmt.Fprint(w, `{"id":"action-1"}`)
					default:
						t.Errorf("unexpected mutation/request: %s %s", r.Method, r.URL)
						http.Error(w, "unexpected", 500)
					}
				})
				result, err := client.PublishReview(context.Background(), opts)
				if lost != "none" {
					if err == nil {
						t.Fatal("expected lost response failure")
					}
					result, err = client.PublishReview(context.Background(), opts)
				}
				if err != nil {
					t.Fatal(err)
				}
				if result.ActionID != "action-1" || result.ReviewID != opts.ReviewID {
					t.Fatal(result)
				}
				// A successful retry is a local no-op, even if the network is unavailable.
				if _, err := client.PublishReview(context.Background(), opts); err != nil {
					t.Fatal(err)
				}
				wantUploads := 1
				if kind == "text" {
					wantUploads = 0
				}
				if uploads != wantUploads || posts != 1 {
					t.Fatalf("uploads=%d posts=%d", uploads, posts)
				}
				stat, _ := os.Stat(opts.StatePath)
				if stat.Mode().Perm() != 0600 {
					t.Fatal("state must be private")
				}
				if kind == "text" {
					opts.Text += "修改"
				} else {
					os.WriteFile(opts.File, []byte("changed"), 0600)
				}
				if _, err := client.PublishReview(context.Background(), opts); err == nil {
					t.Fatal("must reject changed content for same review ID")
				}
			})
		}
	}
}

func TestReviewUncertainDoesNotBlindlyPost(t *testing.T) {
	opts := reviewOpts(t, "text")
	posts := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, `[]`)
			return
		}
		posts++
		http.Error(w, "timeout", 504)
	})
	if _, err := client.PublishReview(context.Background(), opts); err == nil {
		t.Fatal("expected error")
	}
	if _, err := client.PublishReview(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatal("blind repost", posts)
	}
	opts.RetryUncertain = true
	client.PublishReview(context.Background(), opts)
	if posts != 2 {
		t.Fatal("explicit retry should post", posts)
	}
}

func TestReviewRepeatedTextCreatesNewHistory(t *testing.T) {
	opts := reviewOpts(t, "text")
	var comments []reviewComment
	posts := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(comments)
			return
		}
		posts++
		r.ParseForm()
		comment := reviewComment{ID: fmt.Sprint(posts)}
		comment.Data.Text = r.Form.Get("text")
		comments = append(comments, comment)
		// Drop both responses to test matching against the baseline of old identical text.
		fmt.Fprint(w, `broken`)
	})
	for i := 0; i < 2; i++ {
		opts.ReviewID = fmt.Sprint(i)
		opts.StatePath = filepath.Join(t.TempDir(), "state.json")
		if _, err := client.PublishReview(context.Background(), opts); err == nil {
			t.Fatal("expected lost response")
		}
		result, err := client.PublishReview(context.Background(), opts)
		if err != nil || result.ActionID != fmt.Sprint(i+1) {
			t.Fatal(result, err)
		}
	}
	if posts != 2 {
		t.Fatal(posts)
	}
}

func TestReviewRejectsInvalidInputsWithoutNetwork(t *testing.T) {
	cases := []struct{ kind, text, file string }{
		{"text", " ", ""}, {"text", "hello", "file.png"}, {"image", "hello", "file.png"}, {"audio", "hello", "file.m4a"}, {"image", "", "file.svg"}, {"audio", "", "file.exe"}, {"other", "", ""},
	}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected network access") })
	for _, tc := range cases {
		opts := ReviewOptions{CardID: "card-1", ReviewID: "review", StatePath: filepath.Join(t.TempDir(), "state"), Type: tc.kind, Text: tc.text, File: tc.file}
		if _, err := client.PublishReview(context.Background(), opts); err == nil {
			t.Error(tc)
		}
	}
}

func TestReviewRejectsLinkOrMismatchedAttachment(t *testing.T) {
	for _, att := range []reviewAttachment{
		{ID: "a", Name: "homework-review-image-test-review.png", MIMEType: "image/png"},
		{ID: "a", Name: "homework-review-image-test-review.png", MIMEType: "audio/mp4", IsUpload: true},
	} {
		opts := reviewOpts(t, "image")
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/attachments") {
				t.Error("must not publish invalid attachment")
			}
			json.NewEncoder(w).Encode([]reviewAttachment{att})
		})
		if _, err := client.PublishReview(context.Background(), opts); err == nil {
			t.Fatal("accepted invalid attachment")
		}
	}
}

func TestReviewCommentsPaginates(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			page := make([]reviewComment, 1000)
			for i := range page {
				page[i].ID = fmt.Sprint(i)
			}
			json.NewEncoder(w).Encode(page)
		} else {
			if r.URL.Query().Get("before") != "999" {
				t.Error(r.URL)
			}
			fmt.Fprint(w, `[{"id":"old"}]`)
		}
	})
	comments, err := client.reviewComments(context.Background(), "card-1")
	if err != nil || len(comments) != 1001 || calls != 2 {
		t.Fatal(len(comments), calls, err)
	}
}

func TestReviewRecoversAfterConnectionCloses(t *testing.T) {
	opts := reviewOpts(t, "text")
	var saved reviewComment
	posts := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if saved.ID == "" {
				fmt.Fprint(w, `[]`)
			} else {
				json.NewEncoder(w).Encode([]reviewComment{saved})
			}
			return
		}
		posts++
		r.ParseForm()
		saved.ID = "accepted-before-disconnect"
		saved.Data.Text = r.PostForm.Get("text")
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		conn.Close()
	})
	if _, err := client.PublishReview(context.Background(), opts); err == nil {
		t.Fatal("expected connection error")
	}
	result, err := client.PublishReview(context.Background(), opts)
	if err != nil || posts != 1 || result.ActionID != saved.ID {
		t.Fatal(result, posts, err)
	}
}

func TestReviewLookupFailureNeverPublishes(t *testing.T) {
	for _, kind := range []string{"text", "image", "audio"} {
		t.Run(kind, func(t *testing.T) {
			opts := reviewOpts(t, kind)
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Error("must not POST after failed reconciliation")
				}
				http.Error(w, "offline", 503)
			})
			if _, err := client.PublishReview(context.Background(), opts); err == nil {
				t.Fatal("expected lookup error")
			}
		})
	}
}

func TestReviewOutcomeWorkflow(t *testing.T) {
	for _, tc := range []struct {
		name, current, target          string
		hasErrors, missing, failUpdate bool
	}{
		{"errors", "done", "待改错", true, false, false},
		{"initial-pass", "done", "已完成", false, false, false},
		{"correction-pass", "corrected", "已改错", false, false, false},
		{"correction-fails", "corrected", "待改错", true, false, false},
		{"missing-list", "done", "待改错", true, true, false},
		{"retry-status-only", "corrected", "待改错", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := reviewOpts(t, "text")
			opts.HasErrors = &tc.hasErrors
			if tc.failUpdate {
				opts.Grade = "良"
			}
			posts, updates, creates, gradeUpdates := 0, 0, 0, 0
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "GET /cards/card-1/actions":
					fmt.Fprint(w, `[]`)
				case "POST /cards/card-1/actions/comments":
					posts++
					r.ParseForm()
					var payload reviewPayload
					if err := json.Unmarshal([]byte(strings.TrimPrefix(r.FormValue("text"), ReviewPrefix)), &payload); err != nil || payload.HasErrors == nil || *payload.HasErrors != tc.hasErrors {
						t.Errorf("outcome missing from protocol: %v, %v", payload, err)
					}
					fmt.Fprint(w, `{"id":"review-action"}`)
				case "GET /cards/card-1":
					if r.URL.Query().Get("fields") == "desc" {
						fmt.Fprint(w, `{"desc":"原题"}`)
						return
					}
					if r.URL.Query().Get("fields") != "idBoard,idList" {
						t.Error("must request board and list")
					}
					fmt.Fprintf(w, `{"idBoard":"board","idList":%q}`, tc.current)
				case "GET /boards/board/lists":
					if tc.missing {
						fmt.Fprint(w, `[]`)
					} else {
						fmt.Fprint(w, `[{"id":"done","name":"已完成"},{"id":"correction","name":"待改错"},{"id":"corrected","name":"已改错"}]`)
					}
				case "POST /lists":
					creates++
					if r.URL.Query().Get("name") != tc.target || r.URL.Query().Get("idBoard") != "board" {
						t.Error(r.URL)
					}
					fmt.Fprint(w, `{"id":"new-list"}`)
				case "PUT /cards/card-1":
					if r.URL.Query().Has("desc") {
						gradeUpdates++
						fmt.Fprint(w, `{}`)
						return
					}
					updates++
					want := map[string]string{"待改错": "correction", "已完成": "done", "已改错": "corrected"}[tc.target]
					if tc.missing {
						want = "new-list"
					}
					if r.URL.Query().Get("idList") != want || r.URL.Query().Get("dueComplete") != fmt.Sprint(!tc.hasErrors) {
						t.Error(r.URL)
					}
					if tc.failUpdate && updates == 1 {
						http.Error(w, "temporary failure", 500)
						return
					}
					fmt.Fprint(w, `{}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				}
			})
			result, err := client.PublishReview(context.Background(), opts)
			if tc.failUpdate {
				if err == nil {
					t.Fatal("must report incomplete status update")
				}
				result, err = client.PublishReview(context.Background(), opts)
			}
			if err != nil || result.Status != tc.target {
				t.Fatalf("%+v %v", result, err)
			}
			if _, err := client.PublishReview(context.Background(), opts); err != nil {
				t.Fatal(err)
			}
			if tc.failUpdate && gradeUpdates != 1 {
				t.Fatalf("grade must not replay on status retry: %d", gradeUpdates)
			}
			wantUpdates := 1
			if tc.failUpdate {
				wantUpdates = 2
			}
			if posts != 1 || updates != wantUpdates || (tc.missing && creates != 1) {
				t.Fatalf("posts=%d updates=%d creates=%d", posts, updates, creates)
			}
			different := !tc.hasErrors
			opts.HasErrors = &different
			if _, err := client.PublishReview(context.Background(), opts); err == nil {
				t.Fatal("must reject changed outcome for the same review ID")
			}
		})
	}
}
