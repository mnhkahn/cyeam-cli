// Package homework implements the cyeam.com homework review API client.
package homework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://www.cyeam.com"
	defaultTimeout     = 175 * time.Second
	defaultMaxBodySize = 16 << 20
)

type ReviewRequest struct {
	Kind     string
	Count    int
	Words    []string
	Filename string
	Image    []byte
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (c *Client) Review(ctx context.Context, input ReviewRequest) (json.RawMessage, error) {
	fields := map[string]string{"kind": input.Kind, "count": strconv.Itoa(input.Count)}
	if input.Kind == "dictation" {
		fields["words"] = strings.Join(input.Words, "\n")
	}
	return c.upload(ctx, "/api/homework/review", fields, "image", input.Filename, input.Image)
}

func (c *Client) Regrade(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return c.postJSON(ctx, "/api/homework/regrade", input)
}

func (c *Client) Reading(ctx context.Context, reference, filename string, audio []byte) (json.RawMessage, error) {
	return c.upload(ctx, "/api/homework/reading", map[string]string{"reference": reference}, "audio", filename, audio)
}

func (c *Client) ReadingRegrade(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return c.postJSON(ctx, "/api/homework/reading/regrade", input)
}

func (c *Client) postJSON(ctx context.Context, path string, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) upload(ctx context.Context, path string, fields map[string]string, field, filename string, data []byte) (json.RawMessage, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, err
		}
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.do(req)
}

func (c *Client) do(req *http.Request) (json.RawMessage, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cyeam-cli/1.0")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("homework request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read homework response: %w", err)
	}
	if len(body) > defaultMaxBodySize {
		return nil, fmt.Errorf("homework response exceeds %d bytes", defaultMaxBodySize)
	}
	var parsed struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("invalid homework JSON response")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(parsed.Error)
		if message == "" {
			message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("homework service error: %s", message)
	}
	return json.RawMessage(bytes.TrimSpace(body)), nil
}
