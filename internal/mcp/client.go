package mcp

import (
	"context"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const DefaultServerURL = "https://cyeam-wiki-mcp-production.up.railway.app/mcp"

type NewsItem struct {
	Link        string `json:"link"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	CreateTime  int64  `json:"create_time"`
}

type Client struct {
	serverURL string
}

func NewClient(serverURL string) *Client {
	if serverURL == "" {
		serverURL = DefaultServerURL
	}
	return &Client{serverURL: serverURL}
}

func (c *Client) GetNews(ctx context.Context, toolName string, date string) ([]NewsItem, error) {
	transportClient, err := transport.NewStreamableHTTP(c.serverURL)
	if err != nil {
		return nil, err
	}

	mcpClient := client.NewClient(transportClient)
	defer mcpClient.Close()

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		return nil, err
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = map[string]interface{}{
		"date": date,
	}

	result, err := mcpClient.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, content := range result.Content {
		if link, ok := content.(mcp.ResourceLink); ok {
			ts, desc := parseDescription(link.Description)
			item := NewsItem{
				Link:        link.URI,
				Title:       link.Title,
				Description: desc,
				Image:       "",
				CreateTime:  ts,
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func parseDescription(raw string) (int64, string) {
	parts := strings.SplitN(raw, "|||", 2)
	if len(parts) == 2 {
		if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			return ts, strings.TrimSpace(parts[1])
		}
	}
	return 0, raw
}
