package cyeam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/client"
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

func (s *Service) NewsList(ctx context.Context, from, to string) ([]byte, error) {
	params := map[string]string{}
	if from != "" {
		params["from"] = from
	}
	if to != "" {
		params["to"] = to
	}
	return s.client.GetJSON(ctx, "/api/geek/news", params)
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
