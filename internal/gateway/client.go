package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type Error struct {
	Code        string
	Message     string
	OperationID string
}

func (e *Error) Error() string     { return e.Code + ": " + e.Message }
func (e *Error) ErrorCode() string { return e.Code }

const DefaultURL = "https://homework.cyeam.com"

type Config struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
	Audience string `json:"audience"`
}
type Client struct {
	Base   string
	HTTP   *http.Client
	Token  func(context.Context, string) (string, error)
	config *Config
}

func New() (*Client, error) {
	base := os.Getenv("CYEAM_GATEWAY_URL")
	if base == "" {
		base = DefaultURL
	}
	return NewAt(base)
}
func NewAt(base string) (*Client, error) {
	u, e := url.Parse(base)
	if e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("invalid gateway origin")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost")) {
		return nil, fmt.Errorf("gateway requires HTTPS")
	}
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Token: auth.GetResourceAccessToken}, nil
}
func (c *Client) Config(ctx context.Context) (Config, error) {
	if c.config != nil {
		return *c.config, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.Base+"/v1/auth/config", nil)
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return Config{}, fmt.Errorf("gateway discovery failed: %w", e)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Config{}, apiError(resp)
	}
	var v Config
	if e = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&v); e != nil {
		return v, e
	}
	expected := "api://" + v.Audience + "/"
	if v.ClientID != "e1f582c1-1568-4347-8c5b-1906164e637f" || !regexp.MustCompile(`^[a-fA-F0-9-]{36}$`).MatchString(v.Audience) || !strings.HasPrefix(v.Scope, expected) || strings.ContainsAny(v.Scope, " \t\r\n") {
		return v, fmt.Errorf("gateway returned an unexpected login resource")
	}
	c.config = &v
	return v, nil
}
func (c *Client) Login(ctx context.Context, out, errOut io.Writer) error {
	cfg, e := c.Config(ctx)
	if e != nil {
		return e
	}
	return auth.LoginResource(ctx, cfg.Scope, out, errOut)
}
func apiError(resp *http.Response) error {
	var value struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&value)
	if value.Error == "" {
		value.Error = http.StatusText(resp.StatusCode)
	}
	if value.Code == "" {
		value.Code = "GATEWAY_ERROR"
	}
	return &Error{Code: value.Code, Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, value.Error)}
}
func (c *Client) Do(ctx context.Context, method, path string, value any, idem string) ([]byte, error) {
	cfg, e := c.Config(ctx)
	if e != nil {
		return nil, e
	}
	token, e := c.Token(ctx, cfg.Scope)
	if e != nil {
		return nil, e
	}
	var payload []byte
	if value != nil {
		payload, e = json.Marshal(value)
		if e != nil {
			return nil, e
		}
	}
	req, e := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(payload))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return nil, fmt.Errorf("gateway request failed: %w", e)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 3*1024*1024))
}
func (c *Client) Device(ctx context.Context, id string) (string, error) {
	if id != "" {
		if !regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`).MatchString(id) {
			return "", fmt.Errorf("invalid --device")
		}
		return id, nil
	}
	raw, e := c.Do(ctx, "GET", "/v1/devices", nil, "")
	if e != nil {
		return "", e
	}
	var result struct {
		Devices []struct {
			ID     string `json:"id"`
			Online bool   `json:"online"`
		} `json:"devices"`
	}
	if e = json.Unmarshal(raw, &result); e != nil {
		return "", e
	}
	var online []string
	for _, d := range result.Devices {
		if d.Online {
			online = append(online, d.ID)
		}
	}
	if len(online) != 1 {
		return "", fmt.Errorf("select --device explicitly: found %d online devices; run cyeam device list", len(online))
	}
	return online[0], nil
}
func (c *Client) Call(ctx context.Context, id, name string, args any, idem string) (json.RawMessage, error) {
	if idem == "" {
		b := make([]byte, 16)
		if _, e := rand.Read(b); e != nil {
			return nil, e
		}
		idem = hex.EncodeToString(b)
	}
	raw, e := c.Do(ctx, "POST", "/v1/devices/"+id+"/call", map[string]any{"name": name, "arguments": args}, idem)
	if e != nil {
		return nil, fmt.Errorf("submission failed or is uncertain; retry with --idempotency-key %s: %w", idem, e)
	}
	var accepted struct {
		ID string `json:"operation_id"`
	}
	if e = json.Unmarshal(raw, &accepted); e != nil {
		return nil, e
	}
	if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(accepted.ID) {
		return nil, fmt.Errorf("invalid operation ID")
	}
	for {
		raw, e = c.Do(ctx, "GET", "/v1/operations/"+accepted.ID, nil, "")
		if e != nil {
			return nil, e
		}
		var op struct {
			Status string          `json:"status"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if e = json.Unmarshal(raw, &op); e != nil {
			return nil, e
		}
		if op.Status == "succeeded" {
			return op.Result, nil
		}
		if op.Status == "failed" {
			code := "DEVICE_OPERATION_FAILED"
			var failure struct {
				Code int `json:"code"`
			}
			_ = json.Unmarshal(op.Error, &failure)
			if failure.Code == -32006 {
				code = "SCREEN_CAPTURE_UNAVAILABLE"
			}
			if failure.Code == -32008 {
				code = "SPEECH_FAILED"
			}
			return nil, &Error{Code: code, OperationID: accepted.ID, Message: fmt.Sprintf("operation %s failed and was not retried: %s", accepted.ID, op.Error)}
		}
		select {
		case <-ctx.Done():
			return nil, &Error{Code: "OPERATION_TIMEOUT", OperationID: accepted.ID, Message: fmt.Sprintf("inspect with cyeam device operation %s; retry only with --idempotency-key %s: %v", accepted.ID, idem, ctx.Err())}
		case <-time.After(300 * time.Millisecond):
		}
	}
}
