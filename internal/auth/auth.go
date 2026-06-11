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
	scopes    = "Files.ReadWrite User.Read"
)

func openBrowser(url string) error {
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

func Login(ctx context.Context, out io.Writer) error {
	dcResp, err := requestDeviceCode(ctx)
	if err != nil {
		return fmt.Errorf("request device code: %w", err)
	}

	fmt.Fprintf(out, "Opening browser for sign in...\n")
	fmt.Fprintf(out, "If browser doesn't open, visit:\n%s\n\nEnter code: %s\n",
		dcResp.VerificationURI, dcResp.UserCode)

	if err := openBrowser(dcResp.VerificationURI); err != nil {
		fmt.Fprintf(out, "Tip: open %s manually and enter code %s\n",
			dcResp.VerificationURI, dcResp.UserCode)
	}

	fmt.Fprintf(out, "Waiting for authentication...\n")

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