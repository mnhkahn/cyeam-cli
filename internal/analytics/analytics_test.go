package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/version"
	"github.com/posthog/posthog-go"
)

func enableForTest(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CYEAM_CLI_NO_ANALYTICS", "DO_NOT_TRACK", "CI"} {
		t.Setenv(key, "")
	}
}

func TestCommandDeliveryIncludesEnvironmentAndGeoIP(t *testing.T) {
	enableForTest(t)
	var body struct {
		APIKey string `json:"api_key"`
		Batch  []struct {
			DistinctID string                 `json:"distinct_id"`
			Event      string                 `json:"event"`
			Properties map[string]interface{} `json:"properties"`
		} `json:"batch"`
	}
	delivered := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/batch/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusOK)
		delivered <- struct{}{}
	}))
	defer server.Close()
	config := clientConfig()
	config.Endpoint = server.URL
	client, err := posthog.NewWithConfig(projectToken, config)
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{client: client, id: "installation-test"}
	tracker.CommandStarted("cyeam mo char detail", version.Current())
	tracker.Close()
	select {
	case <-delivered:
	default:
		t.Fatal("short-lived command did not flush event before exit")
	}
	if body.APIKey != projectToken || len(body.Batch) != 1 {
		t.Fatalf("invalid batch: %+v", body)
	}
	event := body.Batch[0]
	if event.DistinctID != "installation-test" {
		t.Fatalf("distinct_id = %q", event.DistinctID)
	}
	if event.Event != "cli_command_started" {
		t.Fatalf("event = %q", event.Event)
	}
	for key, want := range map[string]interface{}{
		"app":          "cyeam-cli",
		"command_path": "cyeam mo char detail", "command": "mo", "subcommand": "char detail",
		"os": runtime.GOOS, "architecture": runtime.GOARCH, "cli_version": version.Current().Version,
		"$process_person_profile": false,
	} {
		if event.Properties[key] != want {
			t.Errorf("%s = %v, want %v", key, event.Properties[key], want)
		}
	}
	if event.Properties["$geoip_disable"] == true {
		t.Error("GeoIP is disabled")
	}
	if _, exists := event.Properties["$is_server"]; exists {
		t.Error("CLI is marked as a server")
	}
	if event.Properties["$os"] == nil {
		t.Error("missing operating system")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		if event.Properties["$os_version"] == nil {
			t.Error("missing system version")
		}
	}
}

func TestOptOutDoesNotInitializeClientOrWriteID(t *testing.T) {
	for _, key := range []string{"CYEAM_CLI_NO_ANALYTICS", "DO_NOT_TRACK", "CI"} {
		t.Run(key, func(t *testing.T) {
			enableForTest(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv(key, "1")
			tracker := New()
			tracker.CommandStarted("cyeam version", version.Current())
			tracker.Close()
			if tracker.client != nil {
				t.Fatal("initialized client after opt-out")
			}
			if _, err := os.Stat(filepath.Join(home, ".cyeam")); !os.IsNotExist(err) {
				t.Fatal("opt-out wrote state")
			}
		})
	}
}

func TestInstallationIDIsStableAcrossConcurrentStarts(t *testing.T) {
	dir := t.TempDir()
	const n = 12
	ids := make(chan string, n)
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() { defer wg.Done(); ids <- installationID(dir) }()
	}
	wg.Wait()
	close(ids)
	want := installationID(dir)
	if !validID([]byte(want)) {
		t.Fatal("invalid installation ID")
	}
	for id := range ids {
		if id != want {
			t.Fatal("concurrent starts generated different IDs")
		}
	}
	if other := installationID(t.TempDir()); other == want {
		t.Fatal("installations share an ID")
	}
}

func TestCloseIsBoundedWhenServerStalls(t *testing.T) {
	enableForTest(t)
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()
	defer close(release)
	config := clientConfig()
	config.Endpoint = server.URL
	config.BatchUploadTimeout = 80 * time.Millisecond
	config.ShutdownTimeout = 120 * time.Millisecond
	client, err := posthog.NewWithConfig(projectToken, config)
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{client: client, id: "timeout-test"}
	tracker.CommandStarted("cyeam version", version.Current())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request not started")
	}
	start := time.Now()
	tracker.Close()
	if time.Since(start) > time.Second {
		t.Fatal("shutdown blocked on analytics")
	}
}

type deliveryResult struct{ result chan error }

func (d deliveryResult) Success(posthog.APIMessage)              { d.result <- nil }
func (d deliveryResult) Failure(_ posthog.APIMessage, err error) { d.result <- err }

// This opt-in check sends a single clearly marked event to the real project.
func TestLivePostHogCapture(t *testing.T) {
	if os.Getenv("CYEAM_TEST_POSTHOG_LIVE") != "1" {
		t.Skip("set CYEAM_TEST_POSTHOG_LIVE=1 to send one live test event")
	}
	enableForTest(t)
	result := make(chan error, 1)
	config := clientConfig()
	config.Callback = deliveryResult{result: result}
	config.DefaultEventProperties = posthog.NewProperties().Set("integration_test", true)
	client, err := posthog.NewWithConfig(projectToken, config)
	if err != nil {
		t.Fatal(err)
	}
	tracker := &Tracker{client: client, id: randomID()}
	tracker.CommandStarted("cyeam version", version.Current())
	tracker.Close()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("PostHog did not accept the event: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("no delivery confirmation")
	}
}
