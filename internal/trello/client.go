// Package trello provides the Trello REST operations exposed by the CLI.
package trello

import (
	"bytes"
	"context"
	"encoding/json"
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

// AttachmentDownload contains a card attachment and the metadata needed to
// interpret its bytes.
type AttachmentDownload struct {
	ID          string
	Name        string
	MIMEType    string
	ContentType string
	Data        []byte
}

type attachment struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	MIMEType string              `json:"mimeType"`
	URL      string              `json:"url"`
	Previews []attachmentPreview `json:"previews"`
}

// attachmentPreview 是 Trello 为上传图片生成的缩放版本，通常是 WebP，
// 体积比原图小一到两个数量级。
type attachmentPreview struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"bytes"`
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

// download fetches bytes from a Trello download endpoint. Trello's attachment
// download endpoint rejects key/token query parameters (401), so it requires
// an OAuth authorization header instead.
func (c *Client) download(ctx context.Context, endpoint string, maxBytes int64) ([]byte, string, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(baseURL + "/" + strings.TrimLeft(endpoint, "/"))
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf(
		`OAuth oauth_consumer_key="%s", oauth_token="%s"`,
		escapeOAuthHeaderValue(c.APIKey),
		escapeOAuthHeaderValue(c.Token),
	))
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, "", fmt.Errorf("Trello attachment download %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("Trello attachment exceeds %d byte limit", maxBytes)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func escapeOAuthHeaderValue(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

func (c *Client) Boards(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "members/me/boards", url.Values{"fields": {"id,name,closed,url"}})
}

func (c *Client) Board(ctx context.Context, boardID string) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID, url.Values{"fields": {"id,name,desc,closed,url,idOrganization,dateLastActivity"}})
}

func (c *Client) CreateBoard(ctx context.Context, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPost, "boards/", fields, nil, "")
}

func (c *Client) UpdateBoard(ctx context.Context, boardID string, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPut, "boards/"+boardID, fields, nil, "")
}

func (c *Client) DeleteBoard(ctx context.Context, boardID string) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, "boards/"+boardID, nil, nil, "")
}

func (c *Client) Lists(ctx context.Context, boardID string) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID+"/lists", url.Values{"fields": {"id,name,closed,pos"}})
}

func (c *Client) List(ctx context.Context, listID string) ([]byte, error) {
	return c.get(ctx, "lists/"+listID, url.Values{"fields": {"id,name,closed,pos,idBoard"}})
}

func (c *Client) CreateList(ctx context.Context, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPost, "lists", fields, nil, "")
}

func (c *Client) UpdateList(ctx context.Context, listID string, fields url.Values) ([]byte, error) {
	return c.request(ctx, http.MethodPut, "lists/"+listID, fields, nil, "")
}

// ArchiveList 归档一个 List。Trello REST 不支持真删除 List，只能把 closed 置为 true。
func (c *Client) ArchiveList(ctx context.Context, listID string) ([]byte, error) {
	return c.request(ctx, http.MethodPut, "lists/"+listID+"/closed", url.Values{"value": {"true"}}, nil, "")
}

func (c *Client) ListCards(ctx context.Context, listID string) ([]byte, error) {
	return c.get(ctx, "lists/"+listID+"/cards", cardFields())
}

func (c *Client) BoardCards(ctx context.Context, boardID string) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID+"/cards", cardFields())
}

func (c *Client) BoardStatusChanges(ctx context.Context, boardID, since, before string, limit int) ([]byte, error) {
	return c.get(ctx, "boards/"+boardID+"/actions", url.Values{
		"filter":               {"updateCard:idList"},
		"since":                {since},
		"before":               {before},
		"limit":                {fmt.Sprint(limit)},
		"memberCreator":        {"true"},
		"memberCreator_fields": {"fullName,username"},
	})
}

func (c *Client) Card(ctx context.Context, cardID string) ([]byte, error) {
	return c.get(ctx, "cards/"+cardID, cardFields())
}

func (c *Client) DeleteCard(ctx context.Context, cardID string) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, "cards/"+cardID, nil, nil, "")
}

func (c *Client) Attachments(ctx context.Context, cardID string) ([]byte, error) {
	return c.get(ctx, "cards/"+cardID+"/attachments", nil)
}

// DownloadAttachment fetches an uploaded attachment through Trello's
// authenticated download endpoint. maxBytes must be positive.
func (c *Client) DownloadAttachment(ctx context.Context, cardID, attachmentID string, maxBytes int64) (AttachmentDownload, error) {
	return c.DownloadAttachmentSized(ctx, cardID, attachmentID, 0, maxBytes)
}

// DownloadAttachmentSized 下载附件：maxWidth > 0 且有预览图时，下载不超过
// maxWidth 宽的最大预览图（WebP，体积小下载快）；maxWidth <= 0 时下载原图。
// maxBytes must be positive.
func (c *Client) DownloadAttachmentSized(ctx context.Context, cardID, attachmentID string, maxWidth int, maxBytes int64) (AttachmentDownload, error) {
	if maxBytes <= 0 {
		return AttachmentDownload{}, fmt.Errorf("max attachment size must be positive")
	}
	metadata, err := c.get(ctx, "cards/"+cardID+"/attachments/"+attachmentID, nil)
	if err != nil {
		return AttachmentDownload{}, err
	}
	var item attachment
	if err := json.Unmarshal(metadata, &item); err != nil {
		return AttachmentDownload{}, fmt.Errorf("decode attachment metadata: %w", err)
	}
	if item.ID == "" || item.Name == "" {
		return AttachmentDownload{}, fmt.Errorf("Trello attachment metadata is incomplete")
	}
	name := item.Name
	endpoint := "cards/" + cardID + "/attachments/" + attachmentID + "/download/" + url.PathEscape(item.Name)
	if maxWidth > 0 {
		if preview, ok := pickAttachmentPreview(item.Previews, maxWidth); ok {
			name = previewFileName(preview, item.Name)
			endpoint = "cards/" + cardID + "/attachments/" + attachmentID + "/previews/" + preview.ID + "/download/" + url.PathEscape(name)
			item.MIMEType = ""
		}
	}
	body, contentType, err := c.download(ctx, endpoint, maxBytes)
	if err != nil {
		return AttachmentDownload{}, err
	}
	if item.MIMEType == "" {
		item.MIMEType = contentType
	}
	return AttachmentDownload{ID: item.ID, Name: name, MIMEType: item.MIMEType, ContentType: contentType, Data: body}, nil
}

// pickAttachmentPreview 选宽度不超过 maxWidth 的最大预览图；都超过时选最小的一张。
func pickAttachmentPreview(previews []attachmentPreview, maxWidth int) (attachmentPreview, bool) {
	var best, smallest attachmentPreview
	for _, p := range previews {
		if p.ID == "" || p.Width <= 0 {
			continue
		}
		if smallest.ID == "" || p.Width < smallest.Width {
			smallest = p
		}
		if p.Width <= maxWidth && p.Width > best.Width {
			best = p
		}
	}
	if best.ID != "" {
		return best, true
	}
	return smallest, smallest.ID != ""
}

// previewFileName 从预览图 URL 里取文件名（一般是原图名换 .webp 后缀）。
func previewFileName(preview attachmentPreview, fallback string) string {
	if u, err := url.Parse(preview.URL); err == nil {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		if name, err := url.PathUnescape(parts[len(parts)-1]); err == nil && name != "" {
			return name
		}
	}
	return fallback
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

// ProcessCardResult 表示卡片附件的下载信息
type ProcessCardResult struct {
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	MimeType    string `json:"mime_type"`
}

// ProcessCard 通过卡片短链接获取所有附件的下载链接，形如
// https://trello.com/1/cards/<card-id>/attachments/<attachment-id>/download/<filename>
// 链接在已登录 Trello 的浏览器里可直接下载；脚本下载请用 DownloadAttachment。
func (c *Client) ProcessCard(ctx context.Context, shortURL string) ([]ProcessCardResult, error) {
	shortID, err := ExtractShortID(shortURL)
	if err != nil {
		return nil, err
	}

	// 直接用 shortID 请求卡片详情（Trello API 支持 shortID）
	cardData, err := c.get(ctx, "cards/"+shortID, url.Values{"fields": {"id,name"}})
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}
	var card struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(cardData, &card); err != nil {
		return nil, fmt.Errorf("decode card: %w", err)
	}
	if card.ID == "" {
		return nil, fmt.Errorf("card not found")
	}

	// 列出附件
	attachmentsData, err := c.get(ctx, "cards/"+card.ID+"/attachments", nil)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	var attachments []attachment
	if err := json.Unmarshal(attachmentsData, &attachments); err != nil {
		return nil, fmt.Errorf("decode attachments: %w", err)
	}

	// 优先使用附件自带的 url 字段；缺失时按固定格式拼接
	results := make([]ProcessCardResult, 0, len(attachments))
	for _, att := range attachments {
		downloadURL := att.URL
		if downloadURL == "" {
			downloadURL = fmt.Sprintf(
				"https://trello.com/1/cards/%s/attachments/%s/download/%s",
				card.ID,
				att.ID,
				url.PathEscape(att.Name),
			)
		}
		results = append(results, ProcessCardResult{
			FileName:    att.Name,
			DownloadURL: downloadURL,
			MimeType:    att.MIMEType,
		})
	}
	return results, nil
}

// ExtractShortID 从 Trello 短链接中提取 8 位 short ID
// 支持格式：https://trello.com/c/abc12345, https://trello.com/c/abc12345/name
func ExtractShortID(shortURL string) (string, error) {
	u, err := url.Parse(shortURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "c" {
		return "", fmt.Errorf("not a Trello card short URL: %s", shortURL)
	}
	shortID := parts[1]
	if len(shortID) != 8 {
		return "", fmt.Errorf("invalid short ID length: %s", shortID)
	}
	return shortID, nil
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
