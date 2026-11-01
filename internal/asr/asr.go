// Package asr implements the cyeam.com speech-to-text API client.
package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultEndpoint     = "https://www.cyeam.com/api/speech-to-text"
	defaultTimeout      = 2 * time.Minute
	defaultMaxFileBytes = 15 << 20 // server limit: 15 MiB
	defaultMaxBodyBytes = 2 << 20  // 2 MiB
)

// Transcriber converts an audio recording to text.
type Transcriber interface {
	Transcribe(ctx context.Context, filename string, audio []byte) (*Result, error)
}

// Client uploads recordings to the speech-to-text endpoint.
type Client struct {
	Endpoint     string
	HTTPClient   *http.Client
	MaxFileBytes int64
	MaxBodyBytes int64
}

// NewClient creates an ASR client with production defaults.
func NewClient() *Client {
	return &Client{
		Endpoint:     defaultEndpoint,
		HTTPClient:   &http.Client{Timeout: defaultTimeout},
		MaxFileBytes: defaultMaxFileBytes,
		MaxBodyBytes: defaultMaxBodyBytes,
	}
}

// audioContentType picks the upload Content-Type from the file extension,
// falling back to content sniffing. The server only accepts audio/* types and
// Go's sniffer reports m4a files as video/mp4, so the extension wins.
func audioContentType(filename string, data []byte) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".amr":
		return "audio/amr"
	case ".3gp":
		return "audio/3gpp"
	case ".opus":
		return "audio/opus"
	default:
		return http.DetectContentType(data)
	}
}

// Result is the recognized transcript.
type Result struct {
	Text string `json:"text"`
}

// Transcribe uploads one audio file and returns its transcript.
func (c *Client) Transcribe(ctx context.Context, filename string, audio []byte) (*Result, error) {
	maxFileBytes := c.MaxFileBytes
	if maxFileBytes <= 0 {
		maxFileBytes = defaultMaxFileBytes
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("audio file must not be empty")
	}
	if int64(len(audio)) > maxFileBytes {
		return nil, fmt.Errorf("audio file exceeds %d bytes", maxFileBytes)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	name := filepath.Base(filename)
	if name == "." || name == "" {
		name = "audio.m4a"
	}
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="audio"; filename="%s"`, name))
	partHeader.Set("Content-Type", audioContentType(name, audio))
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, fmt.Errorf("build ASR request: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return nil, fmt.Errorf("build ASR request: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("build ASR request: %w", err)
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("build ASR request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cyeam-cli/1.0")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ASR request failed (connection or timeout)")
	}
	defer resp.Body.Close()

	maxBodyBytes := c.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxBodyBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read ASR response: %w", err)
	}
	if int64(len(data)) > maxBodyBytes {
		return nil, fmt.Errorf("ASR response exceeds %d bytes", maxBodyBytes)
	}

	var parsed struct {
		Text  string `json:"text"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid ASR JSON response")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(parsed.Error)
		if message == "" {
			message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("ASR service error: %s", message)
	}
	text := strings.TrimSpace(parsed.Text)
	if text == "" {
		return nil, fmt.Errorf("ASR returned no text")
	}
	return &Result{Text: text}, nil
}
