package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/update"
	"github.com/mnhkahn/cyeam-cli/internal/version"
)

type fakeService struct {
	architectureQuery string
	architectureMode  string
	searchQuery       string
	dateSloganDate    string
	dateHolidayDate   string
	roadbookShareBody string
	roadbookGetID     string
	moGuwenText       string
	moGuwenCompose    bool
	moDetailChar      string
	moCompositionChar string
	moComposeChar     string
	moOCRFilename     string
	moOCRBody         string
}

type fakeUpdater struct {
	current version.Info
	result  update.Result
}

func (f *fakeUpdater) Update(ctx context.Context, current version.Info) (update.Result, error) {
	f.current = current
	return f.result, nil
}

func (f *fakeService) AskArchitecture(ctx context.Context, query string, mode string, out io.Writer) error {
	f.architectureQuery = query
	f.architectureMode = mode
	_, _ = io.WriteString(out, "architecture answer")
	return nil
}

func (f *fakeService) Search(ctx context.Context, query string) ([]byte, error) {
	f.searchQuery = query
	return []byte(`{"docs":[]}`), nil
}

func (f *fakeService) DateSlogan(ctx context.Context, date string) ([]byte, error) {
	f.dateSloganDate = date
	return []byte(`{"date":"` + date + `","slogan":"keep going"}`), nil
}

func (f *fakeService) DateHoliday(ctx context.Context, date string) ([]byte, error) {
	f.dateHolidayDate = date
	return []byte(`{"date":"` + date + `","is_holiday":false}`), nil
}

func (f *fakeService) RoadbookShare(ctx context.Context, body []byte) ([]byte, error) {
	f.roadbookShareBody = string(body)
	return []byte(`{"id":"abc123","url":"https://www.cyeam.com/tool/roadbook?id=abc123"}`), nil
}

func (f *fakeService) RoadbookGet(ctx context.Context, id string) ([]byte, error) {
	f.roadbookGetID = id
	return []byte(`{"data":"[]"}`), nil
}

func (f *fakeService) MoGuwen(ctx context.Context, text string, aiCompose bool) ([]byte, error) {
	f.moGuwenText = text
	f.moGuwenCompose = aiCompose
	return []byte(`{"text":"` + text + `"}`), nil
}

func (f *fakeService) MoCharDetail(ctx context.Context, char string) ([]byte, error) {
	f.moDetailChar = char
	return []byte(`{"char":"` + char + `"}`), nil
}

func (f *fakeService) MoCharComposition(ctx context.Context, char string) ([]byte, error) {
	f.moCompositionChar = char
	return []byte(`{"char":"` + char + `","found":true}`), nil
}

func (f *fakeService) MoCharCompose(ctx context.Context, char string) ([]byte, error) {
	f.moComposeChar = char
	return []byte{1, 2, 3}, nil
}

func (f *fakeService) MoOCR(ctx context.Context, filename string, body []byte) ([]byte, error) {
	f.moOCRFilename = filename
	f.moOCRBody = string(body)
	return []byte(`{"code":0}`), nil
}

func TestAskDefaultsToArchitectureFastMode(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"ask", "系统怎么做限流"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute ask: %v", err)
	}

	if service.architectureQuery != "系统怎么做限流" {
		t.Fatalf("architecture query = %q", service.architectureQuery)
	}
	if service.architectureMode != "fast" {
		t.Fatalf("architecture mode = %q, want fast", service.architectureMode)
	}
	if stdout.String() != "architecture answer" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestVersionPrintsVersionInfo(t *testing.T) {
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Stdout: stdout,
		VersionInfo: func() version.Info {
			return version.Info{
				Version:   "v1.2.3",
				Commit:    "abc123",
				BuildDate: "2026-06-09T12:00:00Z",
				GOOS:      "darwin",
				GOARCH:    "arm64",
			}
		},
	})
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}

	want := "version: v1.2.3\ncommit: abc123\nbuild_date: 2026-06-09T12:00:00Z\ngoos: darwin\ngoarch: arm64\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateDelegatesToUpdater(t *testing.T) {
	stdout := new(bytes.Buffer)
	updater := &fakeUpdater{result: update.Result{
		Updated:    true,
		OldVersion: "v1.0.0",
		NewVersion: "v1.1.0",
	}}
	cmd := NewRootCommand(Dependencies{
		Stdout:  stdout,
		Updater: updater,
		VersionInfo: func() version.Info {
			return version.Info{Version: "v1.0.0"}
		},
	})
	cmd.SetArgs([]string{"update"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute update: %v", err)
	}

	if updater.current.Version != "v1.0.0" {
		t.Fatalf("current version = %q", updater.current.Version)
	}
	if stdout.String() != "updated: v1.0.0 -> v1.1.0\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDateSloganUsesExplicitDate(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"date", "slogan", "2026-06-09"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute date slogan: %v", err)
	}

	if service.dateSloganDate != "2026-06-09" {
		t.Fatalf("date = %q", service.dateSloganDate)
	}
	if stdout.String() != "{\"date\":\"2026-06-09\",\"slogan\":\"keep going\"}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDateHolidayDefaultsToToday(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Service: service,
		Stdout:  stdout,
		Now: func() time.Time {
			return time.Date(2026, 6, 9, 10, 0, 0, 0, time.Local)
		},
	})
	cmd.SetArgs([]string{"date", "holiday"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute date holiday: %v", err)
	}

	if service.dateHolidayDate != "2026-06-09" {
		t.Fatalf("date = %q", service.dateHolidayDate)
	}
}

func TestDateRejectsInvalidDate(t *testing.T) {
	service := &fakeService{}
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: new(bytes.Buffer)})
	cmd.SetArgs([]string{"date", "holiday", "2026-99-09"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestRoadbookShareReadsFileAndPrintsURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roadbook.json")
	if err := os.WriteFile(path, []byte(`[{"name":"A"}]`), 0644); err != nil {
		t.Fatalf("write roadbook: %v", err)
	}

	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"roadbook", "share", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute roadbook share: %v", err)
	}

	if service.roadbookShareBody != `[{"name":"A"}]` {
		t.Fatalf("body = %q", service.roadbookShareBody)
	}
	if stdout.String() != "{\"id\":\"abc123\",\"url\":\"https://www.cyeam.com/tool/roadbook?id=abc123\"}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRoadbookGetUsesID(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"roadbook", "get", "abc123"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute roadbook get: %v", err)
	}

	if service.roadbookGetID != "abc123" {
		t.Fatalf("id = %q", service.roadbookGetID)
	}
	if stdout.String() != "{\"data\":\"[]\"}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestMoGuwenUsesTextAndAIComposeFlag(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"mo", "guwen", "兰亭序", "--ai-compose"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute mo guwen: %v", err)
	}

	if service.moGuwenText != "兰亭序" {
		t.Fatalf("text = %q", service.moGuwenText)
	}
	if !service.moGuwenCompose {
		t.Fatal("ai compose flag not passed")
	}
	if stdout.String() != "{\"text\":\"兰亭序\"}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestMoCharCommandsUseChar(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		got  func(*fakeService) string
	}{
		{
			name: "detail",
			args: []string{"mo", "char", "detail", "之"},
			got:  func(s *fakeService) string { return s.moDetailChar },
		},
		{
			name: "composition",
			args: []string{"mo", "char", "composition", "曦"},
			got:  func(s *fakeService) string { return s.moCompositionChar },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeService{}
			cmd := NewRootCommand(Dependencies{Service: service, Stdout: new(bytes.Buffer)})
			cmd.SetArgs(tc.args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute %s: %v", tc.name, err)
			}

			if tc.got(service) != tc.args[3] {
				t.Fatalf("char = %q", tc.got(service))
			}
		})
	}
}

func TestMoCharComposeWritesPNG(t *testing.T) {
	service := &fakeService{}
	dir := t.TempDir()
	out := filepath.Join(dir, "char.png")
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: new(bytes.Buffer)})
	cmd.SetArgs([]string{"mo", "char", "compose", "曦", "--out", out})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute mo char compose: %v", err)
	}

	if service.moComposeChar != "曦" {
		t.Fatalf("char = %q", service.moComposeChar)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("file = %v", got)
	}
}

func TestMoOCRUploadsImageFile(t *testing.T) {
	service := &fakeService{}
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	if err := os.WriteFile(path, []byte("png-data"), 0644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"mo", "ocr", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute mo ocr: %v", err)
	}

	if service.moOCRFilename != path {
		t.Fatalf("filename = %q", service.moOCRFilename)
	}
	if service.moOCRBody != "png-data" {
		t.Fatalf("body = %q", service.moOCRBody)
	}
	if stdout.String() != "{\"code\":0}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestAskSearchCallsSearchEndpoint(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"ask", "search", "golang 优化"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute ask search: %v", err)
	}

	if service.searchQuery != "golang 优化" {
		t.Fatalf("search query = %q", service.searchQuery)
	}
	if stdout.String() != "{\"docs\":[]}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
