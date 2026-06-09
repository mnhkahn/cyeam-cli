package cyeam

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/client"
)

type Service struct {
	client  *client.Client
	baseURL string
}

func NewService(client *client.Client, baseURL string) *Service {
	return &Service{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *Service) AskArchitecture(ctx context.Context, query string, mode string, out io.Writer) error {
	return s.client.StreamGET(ctx, "/ai/architecture", map[string]string{
		"q":    query,
		"mode": mode,
	}, out)
}

func (s *Service) Search(ctx context.Context, query string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/search/api", map[string]string{"q": query})
}

func (s *Service) DateSlogan(ctx context.Context, date string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/api/date/slogan", map[string]string{"date": date})
}

func (s *Service) DateHoliday(ctx context.Context, date string) ([]byte, error) {
	return s.client.GetJSON(ctx, "/api/date/holiday", map[string]string{"date": date})
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
