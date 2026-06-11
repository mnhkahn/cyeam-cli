package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

const defaultUserAgent = "cyeam-cli/1.0"

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) GetJSON(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path, params), nil)
	if err != nil {
		return nil, err
	}
	return c.doBytes(req)
}

func (c *Client) PostRaw(ctx context.Context, path string, params map[string]string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path, params), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doBytes(req)
}

func (c *Client) StreamGET(ctx context.Context, path string, params map[string]string, out io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path, params), nil)
	if err != nil {
		return err
	}
	applyDefaultHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	if isEventStream(resp.Header.Get("Content-Type")) {
		return streamSSEContent(resp.Body, out)
	}
	_, err = io.Copy(out, resp.Body)
	return err
}

func (c *Client) DownloadBinary(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path, params), nil)
	if err != nil {
		return nil, err
	}
	return c.doBytes(req)
}

func (c *Client) UploadFile(ctx context.Context, path string, field string, filename string, data []byte) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path, nil), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.doBytes(req)
}

func (c *Client) doBytes(req *http.Request) ([]byte, error) {
	applyDefaultHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) url(path string, params map[string]string) string {
	u := c.baseURL + path
	if len(params) == 0 {
		return u
	}
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return u + "?" + values.Encode()
}

func applyDefaultHeaders(req *http.Request) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	excerpt := strings.TrimSpace(string(body))
	if excerpt == "" {
		return fmt.Errorf("http %d %s", resp.StatusCode, resp.Status)
	}
	return fmt.Errorf("http %d %s: %s", resp.StatusCode, resp.Status, excerpt)
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func streamSSEContent(r io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var data []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := writeSSEData(out, data); err != nil {
				return err
			}
			data = nil
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
			data = append(data, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return writeSSEData(out, data)
}

func writeSSEData(out io.Writer, data []string) error {
	if len(data) == 0 {
		return nil
	}
	payload := strings.Join(data, "\n")
	if payload == "[DONE]" {
		return nil
	}
	var event struct {
		Content string `json:"content"`
		Done    bool   `json:"done"`
	}
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("parse SSE data: %w", err)
	}
	if event.Done || event.Content == "" {
		return nil
	}
	_, err := io.WriteString(out, event.Content)
	return err
}
