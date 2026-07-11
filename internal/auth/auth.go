package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	clientID  = "e1f582c1-1568-4347-8c5b-1906164e637f"
	authority = "https://login.microsoftonline.com/consumers"
	scopes    = "Files.ReadWrite User.Read offline_access"
)

// OpenBrowser opens a URL using the platform's default browser.
func OpenBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

// Login runs the OAuth device-code flow. The actionable link and verification
// code go to stdout so a calling agent reading the stream sees them immediately;
// diagnostic progress lines go to stderr. The flow is identical on every
// platform — on machines with a browser we also try to open it as a convenience.
func Login(ctx context.Context, stdout, stderr io.Writer) error {
	fmt.Fprintf(stderr, "[1] requesting device code from Microsoft...\n")
	dcResp, err := requestDeviceCode(ctx)
	if err != nil {
		return fmt.Errorf("request device code: %w", err)
	}
	fmt.Fprintf(stderr, "[2] device code received, user_code=%s\n", dcResp.UserCode)

	fmt.Fprintf(stdout, "Visit:\n%s\n\nEnter code: %s\n",
		dcResp.VerificationURI, dcResp.UserCode)

	// Best-effort: open a browser on machines that have one. On headless
	// servers this fails silently and the user follows the link above manually.
	_ = OpenBrowser(dcResp.VerificationURI)

	fmt.Fprintf(stderr, "[3] waiting for authentication...\n")

	pollCtx := context.WithoutCancel(ctx)
	fmt.Fprintf(stderr, "[4] starting poll loop (interval=%ds)...\n", dcResp.Interval)
	token, err := pollForToken(pollCtx, dcResp.DeviceCode, dcResp.Interval)
	if err != nil {
		return fmt.Errorf("poll for token: %w", err)
	}
	fmt.Fprintf(stderr, "[5] token received\n")

	expiry := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	if err := StoreToken(TokenSet{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       expiry,
	}); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	fmt.Fprintf(stdout, "Login successful.\n")
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
		t := mergeRefreshedToken(token, newToken, time.Now())
		if err := StoreToken(t); err != nil {
			return "", fmt.Errorf("store refreshed token: %w", err)
		}
		return newToken.AccessToken, nil
	}
	return token.AccessToken, nil
}

func mergeRefreshedToken(current TokenSet, refreshed tokenResponse, now time.Time) TokenSet {
	refreshToken := refreshed.RefreshToken
	if refreshToken == "" {
		refreshToken = current.RefreshToken
	}
	return TokenSet{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshToken,
		Expiry:       now.Add(time.Duration(refreshed.ExpiresIn) * time.Second).Unix(),
	}
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
	data.Set("client_type", "public")
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
	data.Set("client_type", "public")

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
	data.Set("client_type", "public")
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
