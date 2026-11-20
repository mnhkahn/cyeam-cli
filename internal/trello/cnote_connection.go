package trello

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/auth"
)

const cnoteConnectionURL = "https://www.cyeam.com/api/cnote/trello-connection"

var (
	boardIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)
	apiKeyPattern  = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
	tokenPattern   = regexp.MustCompile(`^[a-zA-Z0-9]{32,256}$`)
)

func loadCNoteConnection(ctx context.Context) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := auth.GetAccessToken(ctx)
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return Session{}, nil
	}
	if err != nil {
		return Session{}, errors.New("Microsoft session unavailable; run `cyeam login` again")
	}
	return fetchCNoteConnection(ctx, token, nil)
}

func fetchCNoteConnection(ctx context.Context, microsoftToken string, transport http.RoundTripper) (Session, error) {
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cnoteConnectionURL, nil)
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Authorization", "Bearer "+microsoftToken)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Session{}, errors.New("cannot reach CNote to restore your Trello connection; retry later")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return Session{}, errors.New("Microsoft session rejected by CNote; run `cyeam login` again")
	default:
		return Session{}, fmt.Errorf("CNote Trello connection unavailable (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Connected  *bool `json:"connected"`
		Connection struct {
			Credentials struct {
				Key   string `json:"key"`
				Token string `json:"token"`
			} `json:"credentials"`
			BoardID   string `json:"board_id"`
			BoardName string `json:"board_name"`
		} `json:"connection"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 32*1024))
	if decoder.Decode(&result) != nil || result.Connected == nil {
		return Session{}, errors.New("invalid CNote Trello connection response")
	}
	if !*result.Connected {
		return Session{}, nil
	}
	connection := result.Connection
	if !boardIDPattern.MatchString(connection.BoardID) ||
		!apiKeyPattern.MatchString(connection.Credentials.Key) ||
		!tokenPattern.MatchString(connection.Credentials.Token) {
		return Session{}, errors.New("invalid CNote Trello connection data")
	}
	// Keep the restored credentials in memory for this command. The next command
	// resolves them against its current Microsoft login.
	return Session{
		Credentials: Credentials{APIKey: connection.Credentials.Key, Token: connection.Credentials.Token},
		Source:      "cnote",
		BoardID:     connection.BoardID,
		BoardName:   connection.BoardName,
	}, nil
}
