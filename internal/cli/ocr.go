package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
	"github.com/spf13/cobra"
)

type ocrConfigStore struct {
	save         func(paddleocr.Credentials) (string, error)
	load         func() (paddleocr.Credentials, string, error)
	clear        func() error
	saveEndpoint func(string) error
	loadEndpoint func() (string, error)
}

func newOCRCommand() *cobra.Command {
	return newOCRCommandWithStore(ocrConfigStore{paddleocr.StoreCredentials, paddleocr.LoadCredentials, paddleocr.DeleteCredentials, paddleocr.StoreEndpoint, paddleocr.LoadEndpoint})
}
func newOCRCommandWithStore(store ocrConfigStore) *cobra.Command {
	root := &cobra.Command{Use: "ocr", Short: "Manage your PaddleOCR service configuration"}
	config := &cobra.Command{Use: "config", Short: "Save, inspect, or remove your own OCR credentials"}
	var endpoint string
	var tokenStdin, urlOnly bool
	set := &cobra.Command{Use: "set", Short: "Save an OCR endpoint and token", Args: cobra.NoArgs,
		Long: `Save your own PaddleOCR HTTP endpoint and token. The token is read with
hidden terminal input; use --token-stdin for scripts. Credentials are stored in
the system keyring, or a local 0600 file when the keyring is unavailable.
This command saves configuration without sending a request or verifying access.
Use --url-only to save an endpoint first; run config set again to enter the token.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if urlOnly {
				if tokenStdin {
					return fmt.Errorf("--url-only cannot be combined with --token-stdin")
				}
				if strings.TrimSpace(endpoint) == "" {
					return fmt.Errorf("--url is required with --url-only")
				}
				if err := store.saveEndpoint(endpoint); err != nil {
					return err
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"endpoint_saved": true, "token_saved": false})
			}
			if strings.TrimSpace(endpoint) == "" {
				if store.loadEndpoint == nil {
					return fmt.Errorf("--url is required")
				}
				var err error
				endpoint, err = store.loadEndpoint()
				if err != nil {
					return fmt.Errorf("no saved OCR endpoint; provide --url <endpoint>")
				}
			}
			var token string
			var err error
			if tokenStdin {
				var data []byte
				data, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), 16<<10+1))
				if err != nil {
					return fmt.Errorf("read OCR token from stdin failed")
				}
				if len(data) > 16<<10 {
					return fmt.Errorf("OCR token exceeds 16 KiB")
				}
				token = strings.TrimSpace(string(data))
			} else {
				in, ok := cmd.InOrStdin().(*os.File)
				if !ok {
					return fmt.Errorf("use --token-stdin with non-terminal input")
				}
				token, err = readSecretLine(in, cmd.ErrOrStderr(), "Paste your PaddleOCR token (hidden input): ")
				if err != nil {
					return fmt.Errorf("read OCR token failed")
				}
			}
			storage, err := store.save(paddleocr.Credentials{URL: strings.TrimSpace(endpoint), Token: strings.TrimSpace(token)})
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"saved": true, "storage": storage, "environment_override": os.Getenv("OCR_API_URL") != "" || os.Getenv("OCR_API_TOKEN") != ""})
		}}
	set.Flags().StringVar(&endpoint, "url", "", "your PaddleOCR HTTP service endpoint (default: saved endpoint)")
	set.Flags().BoolVar(&urlOnly, "url-only", false, "save the endpoint only; enter the token in a later config set")
	set.Flags().BoolVar(&tokenStdin, "token-stdin", false, "read token from stdin instead of prompting")
	status := &cobra.Command{Use: "status", Short: "Show configuration status without revealing the token", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, source, err := store.load()
			if err != nil && !errors.Is(err, paddleocr.ErrNotConfigured) {
				return err
			}
			envURL, envToken := os.Getenv("OCR_API_URL") != "", os.Getenv("OCR_API_TOKEN") != ""
			effective := "unconfigured"
			if err == nil {
				effective = "stored"
			}
			if envURL && envToken {
				effective = "environment"
			} else if envURL || envToken {
				effective = "incomplete_environment"
			}
			endpointSaved := false
			if store.loadEndpoint != nil {
				_, e := store.loadEndpoint()
				endpointSaved = e == nil
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"endpoint_saved": endpointSaved, "stored": err == nil, "storage": source, "effective_source": effective, "credentials_verified": false, "configured": effective == "stored" || effective == "environment"})
		}}
	clear := &cobra.Command{Use: "clear", Short: "Remove your saved OCR credentials", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := store.clear(); err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"cleared": true, "environment_override": os.Getenv("OCR_API_URL") != "" || os.Getenv("OCR_API_TOKEN") != ""})
		}}
	config.AddCommand(set, status, clear)
	root.AddCommand(config)
	return root
}
