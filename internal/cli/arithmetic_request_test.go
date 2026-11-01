package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

func TestArithmeticTimeoutRetries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fail    int32
		retries int
		calls   int32
		success bool
	}{
		{"recover", 1, 2, 2, true}, {"exhaust", 9, 2, 3, false}, {"disabled", 9, 0, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				if calls.Add(1) <= tc.fail {
					<-r.Context().Done()
					return
				}
				fmt.Fprint(w, `{"result":"ok"}`)
			}))
			defer server.Close()
			client := &paddleocr.Client{URL: server.URL, Token: "test"}
			var log bytes.Buffer
			data, err := recognizeArithmeticWithRetry(context.Background(), client, []byte("image"), 100*time.Millisecond, tc.retries, true, &log)
			if calls.Load() != tc.calls {
				t.Fatalf("calls=%d", calls.Load())
			}
			if tc.success {
				if err != nil || len(data) == 0 {
					t.Fatal(err)
				}
			} else if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			if tc.calls > 1 && !strings.Contains(log.String(), "正在重试") {
				t.Fatal(log.String())
			}
		})
	}
}

func TestArithmeticDoesNotRetryHTTPErrorOrCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	client := &paddleocr.Client{URL: server.URL, Token: "test"}
	var log bytes.Buffer
	_, err := recognizeArithmeticWithRetry(context.Background(), client, []byte("image"), time.Second, 2, false, &log)
	if err == nil || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = recognizeArithmeticWithRetry(ctx, client, []byte("image"), time.Second, 2, false, &log)
	if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}
