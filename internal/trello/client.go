// Package trello provides the Trello REST operations exposed by the CLI.
package trello

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const defaultBaseURL = "https://api.trello.com/1"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
	Token      string
}

func NewDefault() (*Client, error) {
	credentials, _, err := ResolveCredentials()
	if err != nil {
		return nil, err
	}
	return New(credentials), nil
}

func New(credentials Credentials) *Client {
	return &Client{BaseURL: defaultBaseURL, HTTPClient: http.DefaultClient, APIKey: credentials.APIKey, Token: credentials.Token}
}

func (c *Client) Member(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "members/me", url.Values{"fields": {"id,fullName,username"}})
}

func (c *Client) request(ctx context.Context, method, endpoint string, form url.Values, body io.Reader, contentType string) ([]byte, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(baseURL + "/" + strings.TrimLeft(endpoint, "/"))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for key, values := range form {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	q.Set("key", c.APIKey)
	q.Set("token", c.Token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Trello API %s: %s", resp.Status, strings.TrimSpace(string(response)))
	}
	return response, nil
}

func (c *Client) get(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodGet, endpoint, query, nil, "")
}

func (c *Client) Boards(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "members/me/boards", url.Values{"fields": {"id,name,closed,url"}})
}

func (c *Client) Lists(ctx context.Context, boardID string) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID+"/lists", url.Values{"fields": {"id,name,closed,pos"}})
}

func (c *Client) ListCards(ctx context.Context, listID string) ([]byte, error) {
	return c.get(ctx, "lists/"+listID+"/cards", cardFields())
}

func (c *Client) BoardCards(ctx context.Context, boardID string) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID+"/cards", cardFields())
}

func (c *Client) Attachments(ctx context.Context, cardID string) ([]byte, error) {
	return c.get(ctx, "cards/"+cardID+"/attachments", nil)
}

func (c *Client) Actions(ctx context.Context, cardID string, limit int) ([]byte, error) {
	return c.get(ctx, "cards/"+cardID+"/actions", url.Values{"filter": {"all"}, "limit": {fmt.Sprint(limit)}})
}

func cardFields() url.Values {
	return url.Values{"fields": {"id,name,desc,due,dueComplete,idList,closed,url,dateLastActivity"}}
}

func (c *Client) CreateCard(ctx context.Context, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPost, "cards", fields, nil, "")
}

func (c *Client) UpdateCard(ctx context.Context, cardID string, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPut, "cards/"+cardID, fields, nil, "")
}

func (c *Client) CreateWebhook(ctx context.Context, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPost, "webhooks", fields, nil, "")
}

func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, "webhooks/"+webhookID, nil, nil, "")
}

func (c *Client) AttachFile(ctx context.Context, cardID, filename, displayName string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	name := displayName
	if name == "" {
		name = filepath.Base(filename)
	}
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if displayName != "" {
		if err := writer.WriteField("name", displayName); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return c.request(ctx, http.MethodPost, "cards/"+cardID+"/attachments", nil, &body, writer.FormDataContentType())
}
