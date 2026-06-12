package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const graphBase = "https://graph.microsoft.com/v1.0"

type Client struct {
	httpClient *http.Client
	tokenFunc  func(ctx context.Context) (string, error)
}

type Item struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	WebURL               string `json:"webUrl"`
}

type ChildrenResponse struct {
	Value []Item `json:"value"`
}

func NewClient(tokenFunc func(ctx context.Context) (string, error)) *Client {
	return &Client{
		httpClient: http.DefaultClient,
		tokenFunc:  tokenFunc,
	}
}

func (c *Client) do(ctx context.Context, req *http.Request) ([]byte, error) {
	token, err := c.tokenFunc(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("graph API error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) doGet(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

func esc(segments ...string) string {
	enc := make([]string, len(segments))
	for i, s := range segments {
		enc[i] = url.PathEscape(s)
	}
	return strings.Join(enc, "/")
}

func (c *Client) ListFolder(ctx context.Context, folderPath string) ([]Item, error) {
	p := graphBase + "/me/drive/root:/" + url.PathEscape(folderPath) + ":/children?$select=id,name,lastModifiedDateTime,webUrl&$top=50"
	body, err := c.doGet(ctx, p)
	if err != nil {
		return nil, err
	}
	var cr ChildrenResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, err
	}
	return cr.Value, nil
}

func (c *Client) ReadFileByID(ctx context.Context, itemID string) ([]byte, error) {
	p := graphBase + "/me/drive/items/" + url.PathEscape(itemID) + "/content"
	return c.doGet(ctx, p)
}

func (c *Client) ReadFile(ctx context.Context, folderPath, filename string) ([]byte, error) {
	p := graphBase + "/me/drive/root:/" + esc(folderPath, filename) + ":/content"
	return c.doGet(ctx, p)
}

func (c *Client) WriteFile(ctx context.Context, folderPath, filename, contentType string, content []byte) error {
	p := graphBase + "/me/drive/root:/" + esc(folderPath, filename) + ":/content"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, p, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	_, err = c.do(ctx, req)
	return err
}

type ShareLinkResponse struct {
	Link struct {
		WebURL string `json:"webUrl"`
	} `json:"link"`
}

type UserInfo struct {
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

func (c *Client) GetUserInfo(ctx context.Context) (UserInfo, error) {
	p := graphBase + "/me?$select=displayName,mail,userPrincipalName"
	body, err := c.doGet(ctx, p)
	if err != nil {
		return UserInfo{}, err
	}
	var u UserInfo
	if err := json.Unmarshal(body, &u); err != nil {
		return UserInfo{}, err
	}
	return u, nil
}

func (c *Client) CreateShareLink(ctx context.Context, folderPath, filename string) (string, error) {
	p := graphBase + "/me/drive/root:/" + esc(folderPath, filename) + ":/createLink"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p,
		bytes.NewReader([]byte(`{"type":"view","scope":"anonymous"}`)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(ctx, req)
	if err != nil {
		return "", err
	}
	var sl ShareLinkResponse
	if err := json.Unmarshal(resp, &sl); err != nil {
		return "", err
	}
	return sl.Link.WebURL, nil
}
