# Login + Roadbook Upgrade + CNote Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Microsoft login, upgrade roadbook to use OneDrive, and add CNote commands to cyeam-cli.

**Architecture:** OAuth Device Code Flow authenticates against Microsoft, then CLI calls Graph API directly for OneDrive data (identical to frontend). Roadbook share uses cyeam.com backend for short-ID generation. Token stored in system keychain.

**Tech Stack:** Go 1.23, cobra, `zalando/go-keyring`, Microsoft Graph API

---

### Task 1: Add go-keyring dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Install dependency**

```bash
go get github.com/zalando/go-keyring@latest
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add go-keyring dependency"
```
---

### Task 2: Create auth package - keychain storage

**Files:**
- Create: `internal/auth/keychain.go`

- [ ] **Step 1: Create the keychain.go file**

Create directory and file:

```bash
mkdir -p internal/auth
```

```go
package auth

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keychainService = "cyeam-cli"

type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Expiry       int64  `json:"expiry"`
}

func StoreToken(t TokenSet) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return keyring.Set(keychainService, "microsoft_graph", string(data))
}

func LoadToken() (TokenSet, error) {
	data, err := keyring.Get(keychainService, "microsoft_graph")
	if err != nil {
		return TokenSet{}, fmt.Errorf("no token found, run `cyeam login` first")
	}
	var t TokenSet
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return TokenSet{}, fmt.Errorf("unmarshal token: %w", err)
	}
	return t, nil
}

func DeleteToken() error {
	return keyring.Delete(keychainService, "microsoft_graph")
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/keychain.go
git commit -m "feat: add keychain token storage"
```

---

### Task 3: Create auth package - device code flow

**Files:**
- Create: `internal/auth/auth.go`

- [ ] **Step 1: Create auth.go**

```go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	clientID  = "e1f582c1-1568-4347-8c5b-1906164e637f"
	authority = "https://login.microsoftonline.com/consumers"
	scopes    = "Files.ReadWrite User.Read"
)

func Login(ctx context.Context, out io.Writer) error {
	dcResp, err := requestDeviceCode(ctx)
	if err != nil {
		return fmt.Errorf("request device code: %w", err)
	}

	fmt.Fprintf(out, "To sign in, open:\n%s\n\nEnter code: %s\n",
		dcResp.VerificationURI, dcResp.UserCode)
	fmt.Fprintf(out, "Polling for authentication...\n")

	token, err := pollForToken(ctx, dcResp.DeviceCode, dcResp.Interval)
	if err != nil {
		return fmt.Errorf("poll for token: %w", err)
	}

	expiry := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	if err := StoreToken(TokenSet{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       expiry,
	}); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	fmt.Fprintf(out, "Login successful!\n")
	return nil
}

func Logout() error {
	return DeleteToken()
}

func GetAccessToken(ctx context.Context) (string, error) {
	token, err := LoadToken()
	if err != nil {
		return "", err
	}
	if time.Now().Unix() >= token.Expiry-60 {
		newToken, err := refreshToken(ctx, token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("token refresh failed, run `cyeam login`: %w", err)
		}
		expiry := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second).Unix()
		t := TokenSet{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
			Expiry:       expiry,
		}
		if err := StoreToken(t); err != nil {
			return "", fmt.Errorf("store refreshed token: %w", err)
		}
		return newToken.AccessToken, nil
	}
	return token.AccessToken, nil
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func requestDeviceCode(ctx context.Context) (deviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", scopes)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		authority+"/oauth2/v2.0/devicecode",
		strings.NewReader(data.Encode()))
	if err != nil {
		return deviceCodeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return deviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return deviceCodeResponse{}, fmt.Errorf("device code request failed: %s", body)
	}
	var d deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return deviceCodeResponse{}, err
	}
	return d, nil
}

func pollForToken(ctx context.Context, deviceCode string, interval int) (tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)

	for {
		select {
		case <-ctx.Done():
			return tokenResponse{}, ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			authority+"/oauth2/v2.0/token",
			strings.NewReader(data.Encode()))
		if err != nil {
			return tokenResponse{}, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return tokenResponse{}, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			var t tokenResponse
			if err := json.Unmarshal(body, &t); err != nil {
				return tokenResponse{}, err
			}
			return t, nil
		}

		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if errResp.Error == "authorization_pending" || errResp.Error == "slow_down" {
				continue
			}
			if errResp.Error == "expired_token" || errResp.Error == "access_denied" {
				return tokenResponse{}, fmt.Errorf("authentication %s", errResp.Error)
			}
		}
	}
}

func refreshToken(ctx context.Context, refreshToken string) (tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("refresh_token", refreshToken)
	data.Set("scope", scopes)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		authority+"/oauth2/v2.0/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return tokenResponse{}, fmt.Errorf("token refresh failed: %s", body)
	}
	var t tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return tokenResponse{}, err
	}
	return t, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/auth.go
git commit -m "feat: add device code flow and token refresh"
```

---

### Task 4: Create OneDrive Graph API module

**Files:**
- Create: `internal/onedrive/onedrive.go`

- [ ] **Step 1: Create onedrive.go**

```bash
mkdir -p internal/onedrive
```

```go
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
	p := graphBase + "/me/drive/root:/" + url.PathEscape(folderPath) + ":/children?$select=id,name,lastModifiedDateTime&$top=50"
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/onedrive/
git commit -m "feat: add OneDrive Graph API client"
```

---

### Task 5: Add login/logout commands to CLI

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Add login and logout commands**

Add import to root.go:

```go
import (
	"github.com/mnhkahn/cyeam-cli/internal/auth"
)
```

In `NewRootCommand`, after `root.AddCommand(newRoadbookCommand(deps))`, add:

```go
root.AddCommand(newLoginCommand(deps))
root.AddCommand(newLogoutCommand(deps))
```

Add these new functions after existing command functions:

```go
func newLoginCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in with Microsoft account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Login(cmd.Context(), deps.Stdout)
		},
	}
}

func newLogoutCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out and clear stored credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Logout(); err != nil {
				return err
			}
			_, err := deps.Stdout.Write([]byte("Logged out.\n"))
			return err
		},
	}
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat: add login/logout commands"
```

---

### Task 6: Add roadbook list and upgrade roadbook share

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Add roadbook list and share commands**

Add import:

```go
import (
	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"github.com/mnhkahn/cyeam-cli/internal/onedrive"
)
```

Replace the existing `newRoadbookCommand` function with this version that adds `list` and upgrades `share`:

```go
func newRoadbookCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roadbook",
		Short: "Roadbook sharing",
	}
	cmd.AddCommand(newRoadbookListCommand(deps))
	cmd.AddCommand(newRoadbookShareCommand(deps))
	cmd.AddCommand(newRoadbookGetCommand(deps))
	return cmd
}

func newRoadbookListCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List roadbooks from OneDrive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			oc := onedrive.NewClient(auth.GetAccessToken)
			items, err := oc.ListFolder(cmd.Context(), "路书")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No roadbooks found.\n"))
				return err
			}
			fmt.Fprintf(deps.Stdout, "%-40s %s\n", "文件名", "修改时间")
			for _, item := range items {
				fmt.Fprintf(deps.Stdout, "%-40s %s\n", item.Name, item.LastModifiedDateTime[:10])
			}
			return nil
		},
	}
}

func newRoadbookShareCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "share <filename>",
		Short: "Share a roadbook from OneDrive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			if !strings.HasSuffix(filename, ".json") {
				filename += ".json"
			}

			oc := onedrive.NewClient(auth.GetAccessToken)

			// Step 1: Create anonymous share link via OneDrive
			webURL, err := oc.CreateShareLink(cmd.Context(), "路书", filename)
			if err != nil {
				return fmt.Errorf("create share link failed: %w", err)
			}

			// Step 2: Register short ID via cyeam.com backend
			resp, err := deps.Service.RoadbookShare(cmd.Context(), []byte(`{"url":"`+webURL+`"}`))
			if err != nil {
				return fmt.Errorf("register share link failed: %w", err)
			}
			var result struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(resp, &result); err != nil {
				return err
			}

			shareURL := "https://www.cyeam.com/tool/roadbook?id=" + result.ID
			_, err = deps.Stdout.Write([]byte(shareURL + "\n"))
			return err
		},
	}
}

func newRoadbookGetCommand(deps Dependencies) *cobra.Command {
	// Keep existing implementation from current root.go
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a shared roadbook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := deps.Service.RoadbookGet(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, resp)
		},
	}
}
```

Add imports for fmt, strings, encoding/json, and output package if not already present in root.go.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat: add roadbook list and upgrade share with OneDrive"
```

---

### Task 7: Add CNote commands (list, new, append)

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Add cnote command group**

Add a new function `newCnoteCommand` to root.go:

```go
func newCnoteCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cnote",
		Short: "CNote - cloud notes on OneDrive",
	}
	cmd.AddCommand(newCnoteListCommand(deps))
	cmd.AddCommand(newCnoteNewCommand(deps))
	cmd.AddCommand(newCnoteAppendCommand(deps))
	return cmd
}

func newCnoteListCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notes from OneDrive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			oc := onedrive.NewClient(auth.GetAccessToken)
			items, err := oc.ListFolder(cmd.Context(), "Notes")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No notes found.\n"))
				return err
			}
			fmt.Fprintf(deps.Stdout, "%-30s %s\n", "文件名", "修改时间")
			for _, item := range items {
				name := strings.TrimSuffix(item.Name, ".html")
				fmt.Fprintf(deps.Stdout, "%-30s %s\n", name, item.LastModifiedDateTime[:10])
			}
			return nil
		},
	}
}

func newCnoteNewCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "new <title>",
		Short: "Create a new note (content from stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			content, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			if len(bytes.TrimSpace(content)) == 0 {
				return fmt.Errorf("no content provided (stdin is empty)")
			}

			filename := title + ".html"
			oc := onedrive.NewClient(auth.GetAccessToken)
			return oc.WriteFile(cmd.Context(), "Notes", filename, "text/html", content)
		},
	}
}

func newCnoteAppendCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "append <title>",
		Short: "Append content to an existing note (content from stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			filename := title + ".html"

			oc := onedrive.NewClient(auth.GetAccessToken)
			existing, err := oc.ReadFile(cmd.Context(), "Notes", filename)
			if err != nil {
				return fmt.Errorf("read note: %w", err)
			}

			appendContent, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			if len(bytes.TrimSpace(appendContent)) == 0 {
				return fmt.Errorf("no content provided (stdin is empty)")
			}

			combined := append(existing, appendContent...)
			return oc.WriteFile(cmd.Context(), "Notes", filename, "text/html", combined)
		},
	}
}
```

Add imports for `bytes`, `io`, and `strings` if not already present.

In `NewRootCommand`, add the cnote group:

```go
root.AddCommand(newCnoteCommand(deps))
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat: add cnote list/new/append commands"
```

---

### Task 8: Wire deps in main.go and build test

**Files:**
- Modify: `cmd/cyeam/main.go`

- [ ] **Step 1: Verify main.go needs no changes**

Check if the existing main.go already passes deps correctly. The `Dependencies` struct just needs `Service` and `Updater`:

```bash
go build ./...
```

If build fails, fix the issues.

- [ ] **Step 2: Run all existing tests**

```bash
go test ./... -v -count=1
```

Expected: all tests pass. Note: `internal/cli/root_test.go` might need updates if test code references old `newRoadbookCommand` patterns. Check and fix as needed.

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "feat: integrate login, roadbook, and cnote"
```
