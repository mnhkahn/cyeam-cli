// Package paddleocr implements the PaddleOCR serving protocol used by wangxizhi.
package paddleocr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Box [4]float64

func (b Box) Valid() bool {
	for _, v := range b {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return b[2] > b[0] && b[3] > b[1]
}

type Line struct {
	Text       string   `json:"text"`
	Box        Box      `json:"box"`
	Confidence *float64 `json:"confidence,omitempty"`
	Page       int      `json:"page"`
}
type Client struct {
	URL, Token string
	HTTP       *http.Client
}

// Config uses environment variables, an explicitly selected dotenv file, or
// stored user credentials. It never changes the process environment.
func Config(envFile string) (*Client, error) {
	values := map[string]string{}
	if envFile != "" {
		f, err := os.Open(envFile)
		if err != nil {
			return nil, fmt.Errorf("read OCR env file: %w", err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "export "))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			key = strings.TrimSpace(key)
			if !ok || (key != "OCR_API_URL" && key != "OCR_API_TOKEN") {
				continue
			}
			value = strings.TrimSpace(value)
			if strings.HasPrefix(value, "\"") {
				// Do not interpolate shell variables or execute any dotenv content.
				parsed, err := strconv.Unquote(value)
				if err != nil {
					return nil, fmt.Errorf("invalid quoted %s in OCR env file", key)
				}
				value = parsed
			} else if strings.HasPrefix(value, "'") {
				if !strings.HasSuffix(value, "'") || len(value) < 2 {
					return nil, fmt.Errorf("invalid quoted %s in OCR env file", key)
				}
				value = value[1 : len(value)-1]
			} else {
				value, _, _ = strings.Cut(value, " #")
				value = strings.TrimSpace(value)
			}
			values[key] = value
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read OCR env file: %w", err)
		}
	}
	for _, key := range []string{"OCR_API_URL", "OCR_API_TOKEN"} {
		if v := os.Getenv(key); v != "" {
			values[key] = v
		}
	}
	if envFile == "" && values["OCR_API_URL"] == "" && values["OCR_API_TOKEN"] == "" {
		stored, _, err := LoadCredentials()
		if err != nil {
			return nil, err
		}
		values["OCR_API_URL"], values["OCR_API_TOKEN"] = stored.URL, stored.Token
	} else if values["OCR_API_URL"] == "" || values["OCR_API_TOKEN"] == "" {
		return nil, fmt.Errorf("set both OCR_API_URL and OCR_API_TOKEN, or run `cyeam ocr config set --url <endpoint>`")
	}
	if err := validateCredentials(Credentials{URL: values["OCR_API_URL"], Token: values["OCR_API_TOKEN"]}); err != nil {
		return nil, err
	}

	return &Client{URL: values["OCR_API_URL"], Token: values["OCR_API_TOKEN"], HTTP: &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) RecognizeRaw(ctx context.Context, image []byte) ([]byte, error) {
	if strings.TrimSpace(c.URL) == "" || strings.TrimSpace(c.Token) == "" {
		return nil, ErrNotConfigured
	}
	body, _ := json.Marshal(map[string]any{"file": base64.StdEncoding.EncodeToString(image), "fileType": 1, "useDocOrientationClassify": true, "useDocUnwarping": false, "useLayoutDetection": false, "promptLabel": "spotting"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("invalid OCR request")
	}
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	// Transport errors often include URLs; never include the configured endpoint or token.
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("OCR request failed (connection or timeout)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OCR service returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
	if err != nil {
		return nil, fmt.Errorf("read OCR response failed")
	}
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("OCR response exceeds 32 MiB")
	}
	return data, nil
}

func (c *Client) Recognize(ctx context.Context, image []byte) ([]Line, error) {
	data, err := c.RecognizeRaw(ctx, image)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

type recognition struct {
	Texts    []string      `json:"rec_texts"`
	Scores   []*float64    `json:"rec_scores"`
	Polys    [][][]float64 `json:"rec_polys"`
	Boxes    [][]float64   `json:"rec_boxes"`
	Spotting *recognition  `json:"spotting_res"`
	Blocks   []struct {
		Text string    `json:"block_content"`
		Box  []float64 `json:"block_bbox"`
	} `json:"parsing_res_list"`
}
type page struct {
	Pruned   recognition `json:"prunedResult"`
	Markdown struct {
		Text string `json:"text"`
	} `json:"markdown"`
}

func Parse(data []byte) ([]Line, error) {
	var response struct {
		ErrorCode *int            `json:"errorCode"`
		Result    json.RawMessage `json:"result"`
	}
	if json.Unmarshal(data, &response) != nil {
		return nil, fmt.Errorf("invalid OCR JSON response")
	}
	if response.ErrorCode != nil && *response.ErrorCode != 0 {
		return nil, fmt.Errorf("OCR service error code %d", *response.ErrorCode)
	}
	// Also accept the unwrapped result, useful for replaying OCR evidence offline.
	result := response.Result
	if len(result) == 0 {
		result = data
	}
	var pages struct {
		Layout []page `json:"layoutParsingResults"`
		OCR    []page `json:"ocrResults"`
	}
	if json.Unmarshal(result, &pages) != nil {
		return nil, fmt.Errorf("invalid OCR result")
	}
	all := pages.Layout
	if len(all) == 0 {
		all = pages.OCR
	}
	lines := []Line{}
	for i, p := range all {
		rec := p.Pruned
		if rec.Spotting != nil && len(rec.Spotting.Texts) > 0 {
			rec = *rec.Spotting
		}
		if len(rec.Polys) > 0 && len(rec.Polys) != len(rec.Texts) {
			return nil, fmt.Errorf("OCR text and polygon counts differ")
		}
		if len(rec.Polys) == 0 && len(rec.Boxes) > 0 && len(rec.Boxes) != len(rec.Texts) {
			return nil, fmt.Errorf("OCR text and box counts differ")
		}
		if len(rec.Scores) != len(rec.Texts) {
			rec.Scores = nil
		}
		for j, text := range rec.Texts {
			line := Line{Text: text, Page: i + 1}
			if j < len(rec.Scores) && rec.Scores[j] != nil && *rec.Scores[j] >= 0 && *rec.Scores[j] <= 1 {
				line.Confidence = rec.Scores[j]
			}
			if j < len(rec.Polys) && len(rec.Polys[j]) >= 4 {
				box := Box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
				for _, p := range rec.Polys[j] {
					if len(p) != 2 {
						return nil, fmt.Errorf("invalid OCR polygon")
					}
					box[0] = min(box[0], p[0])
					box[1] = min(box[1], p[1])
					box[2] = max(box[2], p[0])
					box[3] = max(box[3], p[1])
				}
				line.Box = box
			} else if j < len(rec.Boxes) && len(rec.Boxes[j]) == 4 {
				copy(line.Box[:], rec.Boxes[j])
			}
			if strings.TrimSpace(text) != "" {
				lines = append(lines, line)
			}
		}
		if len(rec.Texts) == 0 {
			for _, b := range rec.Blocks {
				line := Line{Text: b.Text, Page: i + 1}
				if len(b.Box) == 4 {
					copy(line.Box[:], b.Box)
				}
				lines = append(lines, line)
			}
			if len(rec.Blocks) == 0 {
				for _, text := range strings.Split(p.Markdown.Text, "\n") {
					if strings.TrimSpace(text) != "" {
						lines = append(lines, Line{Text: text, Page: i + 1})
					}
				}
			}
		}
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("OCR returned no text")
	}
	return lines, nil
}
