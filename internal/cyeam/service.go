package cyeam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
	"github.com/mnhkahn/cyeam-cli/internal/client"
	"github.com/mnhkahn/cyeam-cli/internal/mcp"
	"github.com/mnhkahn/cyeam-cli/internal/news"
)

type Service struct {
	client  *client.Client
	baseURL string
}

func NewService(apiClient *client.Client, baseURL string) *Service {
	return &Service{client: apiClient, baseURL: strings.TrimRight(baseURL, "/")}
}

// DateHoliday reads the same complete daily record rendered by /calendar.
// Calendar calculations and upstream fallbacks belong to the web service.
func (s *Service) DateHoliday(ctx context.Context, date string) ([]byte, error) {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return nil, err
	}
	if t.Year() < 1900 || t.Year() > 2100 {
		return nil, fmt.Errorf("年份须在1900至2100年之间")
	}
	body, err := s.client.GetJSON(ctx, "/api/calendar/dates", map[string]string{"year": strconv.Itoa(t.Year())})
	if err != nil {
		return nil, err
	}
	var dates map[string]json.RawMessage
	if err := json.Unmarshal(body, &dates); err != nil {
		return nil, fmt.Errorf("invalid calendar response: %w", err)
	}
	record, ok := dates[date]
	if !ok {
		return nil, fmt.Errorf("calendar response missing date %s", date)
	}
	var data struct {
		Date      string `json:"date"`
		Type      *int   `json:"type"`
		IsHoliday *bool  `json:"is_holiday"`
		Lunar     string `json:"lunar"`
	}
	if err := json.Unmarshal(record, &data); err != nil {
		return nil, fmt.Errorf("invalid calendar date: %w", err)
	}
	if data.Date != date || data.Type == nil || data.IsHoliday == nil || data.Lunar == "" {
		return nil, fmt.Errorf("incomplete calendar record for %s", date)
	}
	return record, nil
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

func (s *Service) ArithmeticTypes(ctx context.Context, grade int) ([]arithmetic.Type, error) {
	params := map[string]string{}
	if grade != 0 {
		params["grade"] = strconv.Itoa(grade)
	}
	body, err := s.client.GetJSON(ctx, "/api/arithmetic/types", params)
	if err != nil {
		return nil, err
	}
	var result []arithmetic.Type
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) ArithmeticGenerate(ctx context.Context, opts arithmetic.Options) (arithmetic.Exercise, error) {
	params := map[string]string{
		"count":   strconv.Itoa(opts.Count),
		"answers": strconv.FormatBool(opts.Answers),
		"seed":    strconv.FormatInt(opts.Seed, 10),
	}
	if opts.Grade != 0 {
		params["grade"] = strconv.Itoa(opts.Grade)
	}
	if len(opts.Types) != 0 {
		params["type"] = strings.Join(opts.Types, ",")
	}
	body, err := s.client.GetJSON(ctx, "/api/arithmetic/generate", params)
	if err != nil {
		return arithmetic.Exercise{}, err
	}
	var result arithmetic.Exercise
	if err := json.Unmarshal(body, &result); err != nil {
		return arithmetic.Exercise{}, err
	}
	return result, nil
}

func (s *Service) MoGuwen(ctx context.Context, text string, aiCompose bool, font string) ([]byte, error) {
	if font == "" {
		font = "行书"
	}
	params := map[string]string{
		"text": text,
		"font": font,
	}
	if aiCompose {
		params["compose"] = "1"
	}
	return s.client.GetJSON(ctx, "/mo/api/guwen", params)
}

func (s *Service) MoCharDetail(ctx context.Context, char string, font string) ([]byte, error) {
	if font == "" {
		font = "行书"
	}
	return s.client.GetJSON(ctx, "/mo/api/char/detail", map[string]string{
		"char": char,
		"font": font,
	})
}

func (s *Service) MoCharComposition(ctx context.Context, char string, font string) ([]byte, error) {
	if font == "" {
		font = "行书"
	}
	return s.client.GetJSON(ctx, "/mo/api/char/composition", map[string]string{
		"char": char,
		"font": font,
	})
}

func (s *Service) MoCharCompose(ctx context.Context, char string, font string) ([]byte, error) {
	if font == "" {
		font = "行书"
	}
	return s.client.DownloadBinary(ctx, "/mo/api/char/compose", map[string]string{
		"char": char,
		"font": font,
	})
}

func (s *Service) MoOCR(ctx context.Context, filename string, body []byte) ([]byte, error) {
	return s.client.UploadFile(ctx, "/mo/api/ocr", "image", filename, body)
}

type NewsResponse struct {
	Date   string    `json:"date"`
	News   *NewsData `json:"news,omitempty"`
	AINews *NewsData `json:"ai_news,omitempty"`
}

type NewsData struct {
	CreateTime int64          `json:"create_time"`
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

	aiItems, _ := mcpClient.GetAINews(ctx)

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
