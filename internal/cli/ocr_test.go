package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

func TestOCRConfigSetStatusClearWithoutLeakingCredentials(t *testing.T) {
	t.Setenv("OCR_API_URL", "")
	t.Setenv("OCR_API_TOKEN", "")
	var saved paddleocr.Credentials
	exists := false
	store := ocrConfigStore{
		save: func(c paddleocr.Credentials) (string, error) { saved = c; exists = true; return "keyring", nil },
		load: func() (paddleocr.Credentials, string, error) {
			if !exists {
				return paddleocr.Credentials{}, "", paddleocr.ErrNotConfigured
			}
			return saved, "keyring", nil
		},
		clear: func() error { exists = false; saved = paddleocr.Credentials{}; return nil },
	}
	for _, args := range [][]string{{"config", "set", "--url", "https://private-endpoint.invalid/ocr", "--token-stdin"}, {"config", "status"}, {"config", "clear"}, {"config", "status"}} {
		var out bytes.Buffer
		cmd := newOCRCommandWithStore(store)
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader("private-token\n"))
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "private-token") || strings.Contains(out.String(), "private-endpoint") {
			t.Fatal("configuration disclosed")
		}
		if args[1] == "set" && (saved.Token != "private-token" || !strings.Contains(out.String(), `"saved":true`)) {
			t.Fatal("not saved")
		}
	}
	if exists {
		t.Fatal("clear did not remove credentials")
	}
}

func TestOCRSetupEndpointThenToken(t *testing.T) {
	var endpoint string
	var saved paddleocr.Credentials
	store := ocrConfigStore{saveEndpoint: func(s string) error { endpoint = s; return nil }, loadEndpoint: func() (string, error) { return endpoint, nil }, save: func(c paddleocr.Credentials) (string, error) { saved = c; return "keyring", nil }}
	for _, args := range [][]string{{"config", "set", "--url", "https://example.invalid/ocr", "--url-only"}, {"config", "set", "--token-stdin"}} {
		var out bytes.Buffer
		cmd := newOCRCommandWithStore(store)
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader("user-token\n"))
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "user-token") {
			t.Fatal("token leaked")
		}
	}
	if saved.URL != endpoint || saved.Token != "user-token" {
		t.Fatal("saved endpoint was not used")
	}
}
