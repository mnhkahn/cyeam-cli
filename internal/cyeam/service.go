package cyeam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/client"
	"github.com/mnhkahn/cyeam-cli/internal/mcp"
	"github.com/mnhkahn/cyeam-cli/internal/news"
)

const (
	timorHolidayBaseURL = "https://timor.tech"
)

type Service struct {
	client        *client.Client
	holidayClient *client.Client
	baseURL       string
}

func NewService(apiClient *client.Client, baseURL string) *Service {
	return &Service{
		client:        apiClient,
		holidayClient: client.New(timorHolidayBaseURL, nil),
		baseURL:       strings.TrimRight(baseURL, "/"),
	}
}

func (s *Service) DateHoliday(ctx context.Context, date string) ([]byte, error) {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return nil, err
	}
	resp, err := s.holidayClient.GetJSON(ctx, fmt.Sprintf("/api/holiday/year/%d", t.Year()), map[string]string{
		"type":    "Y",
		"weekday": "Y",
	})
	if err != nil {
		return nil, err
	}
	return dateHolidayFromTimor(date, t, resp)
}

func (s *Service) RoadbookShare(ctx context.Context, body []byte) ([]byte, error) {
	resp, err := s.client.PostRaw(ctx, "/api/roadbook/share", nil, body)
	if err != nil {
		return nil, err
	}
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &in); err != nil {
		return nil, err
	}
	out := struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}{
		ID:  in.ID,
		URL: s.baseURL + "/tool/roadbook?id=" + in.ID,
	}
	return json.Marshal(out)
}

func (s *Service) RoadbookGet(ctx context.Context, id string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/api/roadbook/get", map[string]string{"id": id})
}

func (s *Service) MoGuwen(ctx context.Context, text string, aiCompose bool) ([]byte, error) {
	params := map[string]string{
		"text": text,
		"font": "行书",
	}
	if aiCompose {
		params["compose"] = "1"
	}
	return s.client.GetJSON(ctx, "/mo/api/guwen", params)
}

func (s *Service) MoCharDetail(ctx context.Context, char string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/mo/api/char/detail", map[string]string{
		"char": char,
		"font": "行书",
	})
}

func (s *Service) MoCharComposition(ctx context.Context, char string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/mo/api/char/composition", map[string]string{
		"char": char,
		"font": "行书",
	})
}

func (s *Service) MoCharCompose(ctx context.Context, char string) ([]byte, error) {
	return s.client.DownloadBinary(ctx, "/mo/api/char/compose", map[string]string{
		"char": char,
		"font": "行书",
	})
}

func (s *Service) MoOCR(ctx context.Context, filename string, body []byte) ([]byte, error) {
	return s.client.UploadFile(ctx, "/mo/api/ocr", "image", filename, body)
}


type NewsResponse struct {
	Date    string     `json:"date"`
	News    *NewsData  `json:"news,omitempty"`
	AINews  *NewsData  `json:"ai_news,omitempty"`
}

type NewsData struct {
	CreateTime int64           `json:"create_time"`
	News       []mcp.NewsItem `json:"news"`
}

func htmlToMarkdown(html string) string {
	// Remove script/style
	html = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")

	// Links: <a href="url">text</a> -> [text](url)
	html = regexp.MustCompile(`<a[^>]*href=["']([^"']+)["'][^>]*>([^<]+)</a>`).ReplaceAllString(html, "[$2]($1)")

	// Headers: <h1>text</h1> -> # text
	for i := 1; i <= 6; i++ {
		tag := fmt.Sprintf("h%d", i)
		re := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>([^<]+)</%s>`, tag, tag))
		html = re.ReplaceAllString(html, strings.Repeat("#", i)+" $1\n\n")
	}

	// Bold: <b>text</b> -> **text**
	html = regexp.MustCompile(`<b[^>]*>([^<]+)</b>`).ReplaceAllString(html, "**$1**")
	html = regexp.MustCompile(`<strong[^>]*>([^<]+)</strong>`).ReplaceAllString(html, "**$1**")

	// Italic: <i>text</i> or <em>text</em> -> *text*
	html = regexp.MustCompile(`<i[^>]*>([^<]+)</i>`).ReplaceAllString(html, "*$1*")
	html = regexp.MustCompile(`<em[^>]*>([^<]+)</em>`).ReplaceAllString(html, "*$1*")

	// Code: <code>text</code> -> `text`
	html = regexp.MustCompile(`<code[^>]*>([^<]+)</code>`).ReplaceAllString(html, "`$1`")

	// Images: <img src="url" alt="text" ...> -> ![alt](url)
	html = regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*alt=["']([^"']+)["'][^>]*>`).ReplaceAllString(html, "![$2]($1)")
	html = regexp.MustCompile(`<img[^>]*alt=["']([^"']+)["'][^>]*src=["']([^"']+)["'][^>]*>`).ReplaceAllString(html, "![$1]($2)")

	// Paragraphs: <p>text</p> -> text\n\n
	html = regexp.MustCompile(`</p>`).ReplaceAllString(html, "\n\n")

	// Lists: <li>item</li> -> - item
	html = regexp.MustCompile(`<li[^>]*>([^<]+)</li>`).ReplaceAllString(html, "- $1\n")

	// Remove all other HTML tags
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")

	// Clean up whitespace
	html = strings.ReplaceAll(html, "\r", "")
	html = strings.TrimSpace(html)

	// Remove multiple newlines
	html = regexp.MustCompile(`\n{3,}`).ReplaceAllString(html, "\n\n")

	return html
}

func (s *Service) NewsGet(ctx context.Context, date string) ([]byte, error) {
	mcpClient := mcp.NewClient(mcp.DefaultServerURL)

	techItems, err := mcpClient.GetNews(ctx, "tech_news", date)
	if err != nil {
		return nil, fmt.Errorf("tech_news: %w", err)
	}

	// Get AI news from API directly
	var aiItems []mcp.NewsItem
	if aiResp, err := s.client.GetJSON(ctx, "/api/geek/news", map[string]string{"date": date}); err == nil {
		var aiWrapper struct {
			AINews struct {
				News []mcp.NewsItem `json:"news"`
			} `json:"ai_news"`
		}
		if json.Unmarshal(aiResp, &aiWrapper) == nil {
			aiItems = aiWrapper.AINews.News
		}
	}

	// Clean up title prefix (remove [Github Trending], [Hacker News] etc.)
	cleanTitleRe := regexp.MustCompile(`^\[.*?]\s*`)
	for i := range aiItems {
		aiItems[i].Title = cleanTitleRe.ReplaceAllString(aiItems[i].Title, "")
	}
	for i := range techItems {
		techItems[i].Title = cleanTitleRe.ReplaceAllString(techItems[i].Title, "")
	}

	const maxDescLen = 1500
	for i := range techItems {
		// 先从 HTML 描述中提取图片
		if techItems[i].Image == "" {
			techItems[i].Image = news.ExtractImageFromHTML(techItems[i].Description)
		}
		// 没找到再去站点爬
		if techItems[i].Image == "" {
			techItems[i].Image = news.ExtractImage(techItems[i].Link)
		}
		techItems[i].Description = htmlToMarkdown(techItems[i].Description)
		if len(techItems[i].Description) > maxDescLen {
			techItems[i].Description = techItems[i].Description[:maxDescLen] + "..."
		}
	}

	for i := range aiItems {
		// 先从 HTML 描述中提取图片
		if aiItems[i].Image == "" {
			aiItems[i].Image = news.ExtractImageFromHTML(aiItems[i].Description)
		}
		// 没找到再去站点爬
		if aiItems[i].Image == "" {
			aiItems[i].Image = news.ExtractImage(aiItems[i].Link)
		}
		aiItems[i].Description = htmlToMarkdown(aiItems[i].Description)
		if len(aiItems[i].Description) > maxDescLen {
			aiItems[i].Description = aiItems[i].Description[:maxDescLen] + "..."
		}
	}

	resp := NewsResponse{
		Date: date,
		News: &NewsData{
			CreateTime: time.Now().Unix(),
			News:       techItems,
		},
		AINews: &NewsData{
			CreateTime: time.Now().Unix(),
			News:       aiItems,
		},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type timorYearResponse struct {
	Code    int                         `json:"code"`
	Holiday map[string]timorHolidayInfo `json:"holiday"`
	Type    map[string]timorTypeInfo    `json:"type"`
}

type timorHolidayInfo struct {
	Holiday bool   `json:"holiday"`
	Name    string `json:"name"`
	Wage    int    `json:"wage"`
	Date    string `json:"date"`
	After   *bool  `json:"after"`
	Target  string `json:"target"`
	Rest    int    `json:"rest"`
}

type timorTypeInfo struct {
	Type int    `json:"type"`
	Name string `json:"name"`
	Week int    `json:"week"`
}

type dateHolidayOutput struct {
	Date      string `json:"date"`
	IsHoliday bool   `json:"is_holiday"`
	Name      string `json:"name,omitempty"`
	Type      int    `json:"type"`
	Week      int    `json:"week"`
	Wage      int    `json:"wage,omitempty"`
	After     *bool  `json:"after,omitempty"`
	Target    string `json:"target,omitempty"`
	Rest      int    `json:"rest,omitempty"`
}

func dateHolidayFromTimor(date string, t time.Time, body []byte) ([]byte, error) {
	var in timorYearResponse
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, err
	}
	if in.Code != 0 {
		return nil, fmt.Errorf("timor holiday api returned code %d", in.Code)
	}

	out := dateHolidayOutput{
		Date: date,
		Type: 0,
		Week: timorWeek(t),
	}
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		out.IsHoliday = true
		out.Name = weekdayName(t)
		out.Type = 1
	}

	if info, ok := in.Type[date]; ok {
		out.Type = info.Type
		out.Name = info.Name
		if info.Week != 0 {
			out.Week = info.Week
		}
		out.IsHoliday = info.Type == 2
	}

	if info, ok := in.Holiday[t.Format("01-02")]; ok {
		out.IsHoliday = info.Holiday
		if info.Name != "" {
			out.Name = info.Name
		}
		out.Wage = info.Wage
		out.After = info.After
		out.Target = info.Target
		out.Rest = info.Rest
	}

	return json.Marshal(out)
}

func weekdayName(t time.Time) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[int(t.Weekday())]
}

func timorWeek(t time.Time) int {
	week := int(t.Weekday())
	if week == 0 {
		return 7
	}
	return week
}
