package trello

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type connectionTransport func(*http.Request) (*http.Response, error)

func (f connectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchCNoteConnection(t *testing.T) {
	const boardID = "0123456789abcdef01234567"
	response := `{"connected":true,"connection":{"credentials":{"key":"0123456789abcdef0123456789abcdef","token":"testtoken123456789012345678901234"},"board_id":"` + boardID + `","board_name":"Homework"}}`
	transport := connectionTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != cnoteConnectionURL || req.Header.Get("Authorization") != "Bearer microsoft-test" {
			t.Fatal("incorrect CNote connection request")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(response))}, nil
	})
	got, err := fetchCNoteConnection(context.Background(), "microsoft-test", transport)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "cnote" || got.BoardID != boardID || got.BoardName != "Homework" || got.Token == "" {
		t.Fatal("saved connection did not restore")
	}
	response = `{"connected":false}`
	got, err = fetchCNoteConnection(context.Background(), "different-account", connectionTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(response))}, nil
	}))
	if err != nil || got != (Session{}) {
		t.Fatal("an unlinked account inherited a connection")
	}
}

func TestFetchCNoteConnectionDoesNotFollowRedirectsOrEchoBodies(t *testing.T) {
	calls := 0
	transport := connectionTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": {"https://example.net/"}},
			Body:       io.NopCloser(strings.NewReader("secret-placeholder")),
		}, nil
	})
	_, err := fetchCNoteConnection(context.Background(), "microsoft-test", transport)
	if calls != 1 || err == nil || strings.Contains(err.Error(), "secret-placeholder") {
		t.Fatal("connection client followed a redirect or exposed the response body")
	}
}

func TestResolveSessionFallsBackOnlyWhenThereIsNoCloudConnection(t *testing.T) {
	t.Setenv("TRELLO_API_KEY", "")
	t.Setenv("TRELLO_TOKEN", "")
	ctx := context.Background()
	localCalls := 0
	local := func() (Credentials, error) {
		localCalls++
		return Credentials{APIKey: "manual", Token: "local"}, nil
	}
	for _, tc := range []struct {
		session        Session
		err            error
		wantSource     string
		wantLocalCalls int
	}{
		{session: Session{Source: "cnote", BoardID: "saved"}, wantSource: "cnote"},
		{err: errors.New("service unavailable")},
		{wantSource: "stored", wantLocalCalls: 1},
	} {
		got, err := resolveSession(ctx, func(context.Context) (Session, error) {
			return tc.session, tc.err
		}, local)
		if got.Source != tc.wantSource || err != tc.err || localCalls != tc.wantLocalCalls {
			t.Fatal("incorrect cloud or local connection precedence")
		}
	}
}

func TestRestoredTrelloTokenStaysOutOfURLAndErrors(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "fields=id%2CfullName%2Cusername" ||
			r.Header.Get("Authorization") != `OAuth oauth_consumer_key="key", oauth_token="token"` {
			t.Error("restored Trello credentials must only be sent in an authorization header")
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "upstream-secret-placeholder")
	})
	client.cloudConnection = true
	_, err := client.Member(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "upstream-secret-placeholder") {
		t.Fatal("unexpected Trello authentication error")
	}
}
