package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"github.com/mnhkahn/cyeam-cli/internal/onedrive"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/update"
	"github.com/mnhkahn/cyeam-cli/internal/version"
)

type fakeService struct {
	dateHolidayDate   string
	dateHolidayBody   string
	roadbookShareBody string
	roadbookGetID     string
	moGuwenText       string
	moGuwenCompose    bool
	moDetailChar      string
	moCompositionChar string
	moComposeChar     string
	moOCRFilename     string
	moOCRBody         string
	newsListFrom      string
	newsListTo        string
	newsListBody      string
}

type fakeUpdater struct {
	current version.Info
	result  update.Result
}

type fakeOneDrive struct {
	readFolder   string
	readName     string
	readBody     []byte
	writeFolder  string
	writeName    string
	writeContent []byte
}

func (f *fakeOneDrive) ListFolder(ctx context.Context, folderPath string) ([]onedrive.Item, error) {
	return nil, nil
}

func (f *fakeOneDrive) ReadFileByID(ctx context.Context, itemID string) ([]byte, error) {
	return nil, nil
}

func (f *fakeOneDrive) ReadFile(ctx context.Context, folderPath, filename string) ([]byte, error) {
	f.readFolder = folderPath
	f.readName = filename
	return f.readBody, nil
}

func (f *fakeOneDrive) WriteFile(ctx context.Context, folderPath, filename, contentType string, content []byte) error {
	f.writeFolder = folderPath
	f.writeName = filename
	f.writeContent = content
	return nil
}

func (f *fakeOneDrive) CreateShareLink(ctx context.Context, folderPath, filename string) (string, error) {
	return "", nil
}

func (f *fakeOneDrive) GetUserInfo(ctx context.Context) (onedrive.UserInfo, error) {
	return onedrive.UserInfo{}, nil
}

func (f *fakeUpdater) Update(ctx context.Context, current version.Info) (update.Result, error) {
	f.current = current
	return f.result, nil
}

func (f *fakeService) DateHoliday(ctx context.Context, date string) ([]byte, error) {
	f.dateHolidayDate = date
	if f.dateHolidayBody != "" {
		return []byte(f.dateHolidayBody), nil
	}
	return []byte(`{"date":"` + date + `","is_holiday":false,"type":0,"week":2}`), nil
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

func (f *fakeService) NewsList(ctx context.Context, from, to string) ([]byte, error) {
	f.newsListFrom = from
	f.newsListTo = to
	if f.newsListBody != "" {
		return []byte(f.newsListBody), nil
	}
	return []byte(`{"news":{"create_time":1718352000,"news":[],"summary":""},"date":"2026-06-14"}`), nil
}

func init() {
	os.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
}

func envelopeData(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	var env output.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nraw: %s", err, buf.String())
	}
	if !env.OK {
		t.Fatalf("envelope ok false, data: %s", buf.String())
	}
	s, _ := env.Data.(string)
	return s
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
	if envelopeData(t, stdout) != want {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
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
	got := stdout.String()
	if !strings.Contains(got, "updated: v1.0.0 -> v1.1.0") {
		t.Fatalf("stdout = %q, want update message", got)
	}
}

func TestLoginStatusLineMentionsAutoRefreshWhenRefreshTokenExists(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.Local)
	token := auth.TokenSet{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       now.Add(time.Hour).Unix(),
	}

	got := loginStatusLine(token, now)

	if !strings.Contains(got, "access token valid until") {
		t.Fatalf("status line = %q, want access token wording", got)
	}
	if !strings.Contains(got, "auto-refresh enabled") {
		t.Fatalf("status line = %q, want auto-refresh wording", got)
	}
}

func TestDateSloganCommandIsRemoved(t *testing.T) {
	service := &fakeService{}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"date", "slogan", "2026-06-09"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected date slogan to be unsupported")
	}

	if service.dateHolidayDate != "" {
		t.Fatalf("holiday service should not be called, got %q", service.dateHolidayDate)
	}
	if stdout.String() != "" {
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
	want := "日期: 2026-06-09\n星期: 周二\n状态: 工作日\n"
	if envelopeData(t, stdout) != want {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}

func TestDateHolidayFormatsHoliday(t *testing.T) {
	service := &fakeService{
		dateHolidayBody: `{"date":"2026-06-19","is_holiday":true,"name":"端午节","type":2,"week":5,"wage":3}`,
	}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"date", "holiday", "2026-06-19"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute date holiday: %v", err)
	}

	want := "日期: 2026-06-19\n星期: 周五\n状态: 休息日\n名称: 端午节\n薪资倍数: 3\n"
	if envelopeData(t, stdout) != want {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}

func TestDateHolidayFormatsAdjustedWorkday(t *testing.T) {
	service := &fakeService{
		dateHolidayBody: `{"date":"2026-05-09","is_holiday":false,"name":"劳动节后补班","type":3,"week":6,"target":"劳动节"}`,
	}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{Service: service, Stdout: stdout})
	cmd.SetArgs([]string{"date", "holiday", "2026-05-09"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute date holiday: %v", err)
	}

	want := "日期: 2026-05-09\n星期: 周六\n状态: 调休补班\n名称: 劳动节后补班\n目标假期: 劳动节\n"
	if envelopeData(t, stdout) != want {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
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
	if envelopeData(t, stdout) != "{\"id\":\"abc123\",\"url\":\"https://www.cyeam.com/tool/roadbook?id=abc123\"}\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
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
	if envelopeData(t, stdout) != "{\"data\":\"[]\"}\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}

func TestRoadbookCSVParsesCSVAndSavesToOneDrive(t *testing.T) {
	service := &fakeService{}
	od := &fakeOneDrive{}
	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("故宫博物院,北京市东城区景山前街4号,景点,Day1,上午\n国家博物馆,北京市东城区东长安街16号,景点,Day1,下午")

	cmd := NewRootCommand(Dependencies{
		Service:  service,
		Stdout:   stdout,
		OneDrive: func() OneDriveClient { return od },
	})
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"roadbook", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute roadbook csv: %v", err)
	}

	if od.writeFolder != "路书" {
		t.Fatalf("write folder = %q, want 路书", od.writeFolder)
	}
	if od.writeName == "" {
		t.Fatalf("write filename is empty")
	}
	if !strings.HasPrefix(od.writeName, "p_") {
		t.Fatalf("write filename = %q, want p_ prefix", od.writeName)
	}
	var written struct {
		Title string    `json:"title"`
		Items []csvItem `json:"items"`
	}
	if err := json.Unmarshal(od.writeContent, &written); err != nil {
		t.Fatalf("unmarshal od content: %v", err)
	}
	if len(written.Items) != 2 {
		t.Fatalf("items count = %d, want 2", len(written.Items))
	}
	if written.Items[0].Name != "故宫博物院" {
		t.Fatalf("item[0].Name = %q", written.Items[0].Name)
	}
	if written.Items[0].Day != "Day1" {
		t.Fatalf("item[0].Day = %q", written.Items[0].Day)
	}
	if written.Items[1].Name != "国家博物馆" {
		t.Fatalf("item[1].Name = %q", written.Items[1].Name)
	}

	var sharePayload struct {
		Data struct {
			Title string    `json:"title"`
			Items []csvItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(service.roadbookShareBody), &sharePayload); err != nil {
		t.Fatalf("unmarshal share body: %v", err)
	}
	if len(sharePayload.Data.Items) != 2 {
		t.Fatalf("share items = %d", len(sharePayload.Data.Items))
	}

	resp := envelopeData(t, stdout)
	if !strings.Contains(resp, "url") {
		t.Fatalf("stdout = %q, want url field", resp)
	}
}

func TestRoadbookCSVWithCommaInQuotedField(t *testing.T) {
	service := &fakeService{}
	od := &fakeOneDrive{}
	stdout := new(bytes.Buffer)
	stdin := strings.NewReader(`"北京,故宫",北京市东城区景山前街4号,景点,Day1,上午`)

	cmd := NewRootCommand(Dependencies{
		Service:  service,
		Stdout:   stdout,
		OneDrive: func() OneDriveClient { return od },
	})
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"roadbook", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute roadbook csv with commas: %v", err)
	}

	var written struct {
		Items []csvItem `json:"items"`
	}
	if err := json.Unmarshal(od.writeContent, &written); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(written.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(written.Items))
	}
	if written.Items[0].Name != "北京,故宫" {
		t.Fatalf("name = %q, want 北京,故宫", written.Items[0].Name)
	}
	if written.Items[0].Address != "北京市东城区景山前街4号" {
		t.Fatalf("address = %q", written.Items[0].Address)
	}
}

func TestRoadbookCSVWithTitleFlag(t *testing.T) {
	service := &fakeService{}
	od := &fakeOneDrive{}
	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("故宫博物院,北京市东城区景山前街4号,景点,Day1,上午")

	cmd := NewRootCommand(Dependencies{
		Service:  service,
		Stdout:   stdout,
		OneDrive: func() OneDriveClient { return od },
	})
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"roadbook", "csv", "--title", "北京行"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute roadbook csv with title: %v", err)
	}

	var written struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(od.writeContent, &written); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if written.Title != "北京行" {
		t.Fatalf("title = %q, want 北京行", written.Title)
	}
}

func TestRoadbookCSVEmptyInput(t *testing.T) {
	service := &fakeService{}
	od := &fakeOneDrive{}
	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("")

	cmd := NewRootCommand(Dependencies{
		Service:  service,
		Stdout:   stdout,
		OneDrive: func() OneDriveClient { return od },
	})
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"roadbook", "csv"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %q, want empty message", err.Error())
	}
}

func TestParseCSVToItemsSkipsHeaderLine(t *testing.T) {
	input := []byte("名称,地址,类型,日期,备注\n故宫博物院,北京市东城区景山前街4号,景点,Day1,上午")
	items := parseCSVToItems(input)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Name != "故宫博物院" {
		t.Fatalf("name = %q", items[0].Name)
	}
}

func TestParseCSVToItemsWithDayDateNotation(t *testing.T) {
	input := []byte("淄博酒店,淄博市张店区某路,住宿,5.1,入住")
	items := parseCSVToItems(input)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Day != "5.1" {
		t.Fatalf("day = %q, want 5.1", items[0].Day)
	}
	if items[0].Note != "入住" {
		t.Fatalf("note = %q", items[0].Note)
	}
}

func TestParseCSVToItemsWithCommaInQuotedField(t *testing.T) {
	input := []byte(`"北京,故宫博物院",北京市东城区景山前街4号,景点,Day1,上午`)
	items := parseCSVToItems(input)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Name != "北京,故宫博物院" {
		t.Fatalf("name = %q, want 北京,故宫博物院", items[0].Name)
	}
	if items[0].Address != "北京市东城区景山前街4号" {
		t.Fatalf("address = %q", items[0].Address)
	}
}

func TestParseCSVToItemsFiltersInvalidType(t *testing.T) {
	input := []byte("未知地点,地址未知,unknown_type,Day1,测试")
	items := parseCSVToItems(input)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Type != "其他" {
		t.Fatalf("type = %q, want 其他", items[0].Type)
	}
}

func TestParseCSVToItemsLessThan3Columns(t *testing.T) {
	input := []byte("故宫博物院")
	items := parseCSVToItems(input)
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestRoadbookCSVOutputHasValidJSONEnvelope(t *testing.T) {
	service := &fakeService{}
	od := &fakeOneDrive{}
	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("故宫博物院,北京市东城区景山前街4号,景点,Day1,上午")

	cmd := NewRootCommand(Dependencies{
		Service:  service,
		Stdout:   stdout,
		OneDrive: func() OneDriveClient { return od },
	})
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"roadbook", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var env output.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nraw: %s", err, stdout.String())
	}
	if !env.OK {
		t.Fatalf("envelope ok false")
	}
}

func TestRenderRoadbookListTableUsesHyperlinkLabel(t *testing.T) {
	stdout := new(bytes.Buffer)
	rows := []roadbookListRow{{
		Name:     "p_abc123",
		Title:    "成都三日游",
		Modified: "2026-06-10",
	}}

	if err := renderRoadbookListTable(stdout, rows); err != nil {
		t.Fatalf("render table: %v", err)
	}

	got := stdout.String()
	url := "https://www.cyeam.com/tool/roadbook?id=p_abc123"
	link := "\033]8;;" + url + "\033\\链接\033]8;;\033\\"
	if !strings.HasPrefix(got, "┌") {
		t.Fatalf("stdout does not start with table border:\n%s", got)
	}
	if !strings.Contains(got, link) {
		t.Fatalf("stdout missing hyperlink label %q:\n%s", link, got)
	}
	visible := strings.ReplaceAll(got, link, "链接")
	if strings.Contains(visible, url) {
		t.Fatalf("stdout shows raw url outside hyperlink label:\n%s", got)
	}
	for _, part := range []string{"┌", "─", "│", "├", "┼", "└", "\033[35m标题\033[0m", "\033[35m修改时间\033[0m", "\033[35m链接\033[0m"} {
		if !strings.Contains(visible, part) {
			t.Fatalf("stdout missing table part %q:\n%s", part, got)
		}
	}
	if strings.Contains(visible, "+") || strings.Contains(visible, "|") {
		t.Fatalf("stdout still uses ASCII borders:\n%s", got)
	}
}

func TestRenderCnoteListTableUsesSharedTableStyle(t *testing.T) {
	stdout := new(bytes.Buffer)
	url := "https://onedrive.live.com/view.aspx?resid=note-id"
	rows := []cnoteListRow{{
		Name:     "日记",
		Modified: "2026-06-12",
		WebURL:   url,
	}}

	if err := renderCnoteListTable(stdout, rows); err != nil {
		t.Fatalf("render cnote table: %v", err)
	}

	got := stdout.String()
	link := "\033]8;;" + url + "\033\\打开\033]8;;\033\\"
	for _, want := range []string{"┌", "├", "└", "│", "\033[35m文件名\033[0m", "\033[35m修改时间\033[0m", "\033[35m链接\033[0m", "日记", "2026-06-12", link} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q:\n%s", want, got)
		}
	}
	visible := strings.ReplaceAll(got, link, "打开")
	if strings.Contains(visible, url) {
		t.Fatalf("stdout shows raw url outside hyperlink label:\n%s", got)
	}
}

func TestCnoteGetDefaultsToMarkdown(t *testing.T) {
	od := &fakeOneDrive{readBody: []byte(`<h1>标题</h1><p>Hello <strong>world</strong></p>`)}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Stdout: stdout,
		OneDrive: func() OneDriveClient {
			return od
		},
	})
	cmd.SetArgs([]string{"cnote", "get", "日记"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute cnote get: %v", err)
	}
	if od.readFolder != "Notes" || od.readName != "日记.html" {
		t.Fatalf("read = %s/%s", od.readFolder, od.readName)
	}
	if envelopeData(t, stdout) != "# 标题\n\nHello **world**\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}

func TestCnoteGetSupportsTextFormat(t *testing.T) {
	od := &fakeOneDrive{readBody: []byte(`<h2>标题</h2><p>Hello <em>world</em></p>`)}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Stdout: stdout,
		OneDrive: func() OneDriveClient {
			return od
		},
	})
	cmd.SetArgs([]string{"cnote", "get", "日记", "--format", "text"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute cnote get: %v", err)
	}
	if envelopeData(t, stdout) != "标题\n\nHello world\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}

func TestCnoteGetRejectsUnsupportedFormat(t *testing.T) {
	cmd := NewRootCommand(Dependencies{Stdout: new(bytes.Buffer)})
	cmd.SetArgs([]string{"cnote", "get", "日记", "--format", "html"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unsupported format "html"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestFormatCnoteHTMLMarkdown(t *testing.T) {
	html := `<h1>A &amp; B</h1><p>Line<br>Next</p><ul><li>One</li><li><a href="https://example.com">Link</a></li></ul>`
	got := formatCnoteHTML([]byte(html), "markdown")
	want := "# A & B\n\nLine\nNext\n\n- One\n- [Link](https://example.com)"
	if got != want {
		t.Fatalf("markdown = %q", got)
	}
}

func TestFormatCnoteHTMLText(t *testing.T) {
	html := `<h1>A &amp; B</h1><p>Line<br>Next</p><ol><li>One</li><li><a href="https://example.com">Link</a></li></ol>`
	got := formatCnoteHTML([]byte(html), "text")
	want := "A & B\n\nLine\nNext\n\n1. One\n1. Link (https://example.com)"
	if got != want {
		t.Fatalf("text = %q", got)
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
	if envelopeData(t, stdout) != "{\"text\":\"兰亭序\"}\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
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
	if envelopeData(t, stdout) != "{\"code\":0}\n" {
		t.Fatalf("stdout = %q", envelopeData(t, stdout))
	}
}
