package ai

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Model struct {
	Platform      string   `json:"platform"`
	ModelName     string   `json:"model_name"`
	Provider      string   `json:"provider"`
	ModelType     string   `json:"model_type"`
	ContextWindow int      `json:"context_window"`
	ModelSize     string   `json:"model_size"`
	Architecture  string   `json:"architecture"`
	TextRank      int      `json:"text_rank"`
	TextELO       int      `json:"text_elo"`
	CodeRank      int      `json:"code_rank"`
	CodeELO       int      `json:"code_elo"`
	Features      []string `json:"features"`
	Limitations   string   `json:"limitations"`
}

type ModelsResult struct {
	Date   string  `json:"date"`
	Models []Model `json:"models"`
}

const modelsURL = "https://www.cyeam.com/ai/models"

func FetchModels() (*ModelsResult, error) {
	resp, err := http.Get(modelsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetch models: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return parseModels(string(body))
}

func parseModels(htmlContent string) (*ModelsResult, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	result := &ModelsResult{}

	// extract date from page
	result.Date = extractDate(doc)

	// extract models from table rows
	models := extractModels(doc)
	result.Models = models

	return result, nil
}

func extractDate(doc *html.Node) string {
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.TextNode && strings.Contains(n.Data, "更新于") {
			text := strings.TrimSpace(n.Data)
			if idx := strings.Index(text, "更新于 "); idx >= 0 {
				dateStr := text[idx+len("更新于 "):]
				dateStr = strings.TrimSpace(dateStr)
				if len(dateStr) >= 10 {
					found = dateStr[:10]
				}
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

func extractModels(doc *html.Node) []Model {
	var models []Model

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			platform := getAttr(n, "data-platform")
			modelType := getAttr(n, "data-model-type")
			if platform != "" {
				m := Model{
					Platform:  platform,
					ModelType: modelType,
				}
				parseModelRow(n, &m)
				models = append(models, m)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return models
}

func parseModelRow(tr *html.Node, m *Model) {
	// find all td cells
	var tds []*html.Node
	var findTDs func(*html.Node)
	findTDs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			tds = append(tds, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTDs(c)
		}
	}
	findTDs(tr)

	// td[0] - platform name (skip, already have from data attribute)
	// td[1] - model name
	if len(tds) > 1 {
		m.ModelName = cleanupText(tds[1])
	}
	// td[2] - text rank/elo
	if len(tds) > 2 {
		rankStr := getAttr(tds[2], "data-text-rank")
		eloStr := getAttr(tds[2], "data-text-elo")
		if r, err := strconv.Atoi(rankStr); err == nil {
			m.TextRank = r
		}
		if e, err := strconv.Atoi(eloStr); err == nil {
			m.TextELO = e
		}
	}
	// td[3] - code rank/elo
	if len(tds) > 3 {
		rankStr := getAttr(tds[3], "data-code-rank")
		eloStr := getAttr(tds[3], "data-code-elo")
		if r, err := strconv.Atoi(rankStr); err == nil {
			m.CodeRank = r
		}
		if e, err := strconv.Atoi(eloStr); err == nil {
			m.CodeELO = e
		}
	}
	// td[4] - provider
	if len(tds) > 4 {
		m.Provider = cleanupText(tds[4])
	}
	// td[5] - model type (human readable)
	// skip, data-model-type already gives us the key
	// td[6] - context window
	if len(tds) > 6 {
		cw := cleanupText(tds[6])
		if v, err := strconv.Atoi(cw); err == nil {
			m.ContextWindow = v
		}
	}
	// td[7] - model size
	if len(tds) > 7 {
		m.ModelSize = cleanupText(tds[7])
	}
	// td[8] - architecture
	if len(tds) > 8 {
		m.Architecture = cleanupText(tds[8])
	}
	// td[9] - features
	if len(tds) > 9 {
		m.Features = extractFeatures(tds[9])
	}
	// td[10] - limitations
	if len(tds) > 10 {
		m.Limitations = cleanupText(tds[10])
	}
}

func extractFeatures(n *html.Node) []string {
	var features []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				features = append(features, text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return features
}

func cleanupText(n *html.Node) string {
	var text string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text += strings.TrimSpace(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(text)
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}