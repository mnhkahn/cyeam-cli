package phonetic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	defaultBaseURL      = "https://dict.youdao.com/search"
	defaultTimeout      = 15 * time.Second
	defaultMaxBodyBytes = 2 << 20 // 2 MiB
)

// Fetcher looks up phonetic information for a word.
type Fetcher interface {
	Fetch(ctx context.Context, word string) (*Result, error)
}

// Client fetches and parses dictionary pages.
type Client struct {
	BaseURL      string
	HTTPClient   *http.Client
	MaxBodyBytes int64
}

// NewClient creates a dictionary client with production-safe defaults.
func NewClient() *Client {
	return &Client{
		BaseURL:      defaultBaseURL,
		HTTPClient:   &http.Client{Timeout: defaultTimeout},
		MaxBodyBytes: defaultMaxBodyBytes,
	}
}

// Result 音标查询结果
type Result struct {
	Word        string   `json:"word"`
	UKPhonetic  string   `json:"uk_phonetic,omitempty"`
	USPhonetic  string   `json:"us_phonetic,omitempty"`
	Definitions []string `json:"definitions,omitempty"`
}

// Fetch 获取单词音标
func Fetch(ctx context.Context, word string) (*Result, error) {
	return NewClient().Fetch(ctx, word)
}

// Fetch gets phonetic information for one English word.
func (c *Client) Fetch(ctx context.Context, word string) (*Result, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, fmt.Errorf("word must not be empty")
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse dictionary URL: %w", err)
	}
	query := u.Query()
	query.Set("q", word)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	maxBodyBytes := c.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxBodyBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read dictionary response: %w", err)
	}
	if int64(len(body)) > maxBodyBytes {
		return nil, fmt.Errorf("dictionary response exceeds %d bytes", maxBodyBytes)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	result := &Result{
		Word: word,
	}

	// 根据有道词典实际HTML结构解析音标
	// 英式音标和美式音标在 div.baav .pronounce 中
	doc.Find(".baav .pronounce .phonetic").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if i == 0 && text != "" {
			result.UKPhonetic = text
		} else if i == 1 && text != "" {
			result.USPhonetic = text
		}
	})

	// 如果上面没找到，尝试其他选择器
	if result.UKPhonetic == "" && result.USPhonetic == "" {
		doc.Find(".phons .phonetic").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if i == 0 && text != "" {
				result.UKPhonetic = text
			} else if i == 1 && text != "" {
				result.USPhonetic = text
			}
		})
	}

	// 如果上面没找到，尝试另一种选择器
	if result.UKPhonetic == "" && result.USPhonetic == "" {
		doc.Find(".word-phonetic .phonetic").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if i == 0 && text != "" {
				result.UKPhonetic = text
			} else if i == 1 && text != "" {
				result.USPhonetic = text
			}
		})
	}

	// 再尝试另一种结构
	if result.UKPhonetic == "" {
		uk := strings.TrimSpace(doc.Find(".uk .phonetic").Text())
		if uk != "" {
			result.UKPhonetic = uk
		}
	}
	if result.USPhonetic == "" {
		us := strings.TrimSpace(doc.Find(".us .phonetic").Text())
		if us != "" {
			result.USPhonetic = us
		}
	}

	// 获取释义 - 从基础翻译获取，只保留简明释义
	doc.Find("#phrsListTab .trans-container ul li").Each(func(i int, s *goquery.Selection) {
		def := strings.TrimSpace(s.Text())
		// 去除多余的空格和换行
		def = strings.ReplaceAll(def, "\n", " ")
		def = strings.Join(strings.Fields(def), " ")
		if def != "" && !strings.Contains(def, "更多意思请查看") {
			result.Definitions = append(result.Definitions, def)
		}
	})

	// 如果没找到释义，尝试其他选择器
	if len(result.Definitions) == 0 {
		doc.Find(".trans-container ul li").Each(func(i int, s *goquery.Selection) {
			def := strings.TrimSpace(s.Text())
			def = strings.ReplaceAll(def, "\n", " ")
			def = strings.Join(strings.Fields(def), " ")
			if def != "" {
				result.Definitions = append(result.Definitions, def)
			}
		})
	}

	if result.UKPhonetic == "" && result.USPhonetic == "" && len(result.Definitions) == 0 {
		return nil, fmt.Errorf("no phonetic information found for %q", word)
	}

	return result, nil
}

// Format 格式化输出为可读文本
func (r *Result) Format() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("单词: %s\n", r.Word))
	if r.UKPhonetic != "" {
		sb.WriteString(fmt.Sprintf("英式: %s\n", strings.Trim(r.UKPhonetic, "/")))
	}
	if r.USPhonetic != "" {
		sb.WriteString(fmt.Sprintf("美式: %s\n", strings.Trim(r.USPhonetic, "/")))
	}
	if len(r.Definitions) > 0 {
		sb.WriteString("释义:\n")
		for i, def := range r.Definitions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, def))
		}
	}
	return strings.TrimSpace(sb.String())
}
