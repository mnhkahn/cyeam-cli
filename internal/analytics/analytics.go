// Package analytics sends best-effort CLI usage events directly to PostHog.
package analytics

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mnhkahn/cyeam-cli/internal/version"
	"github.com/posthog/posthog-go"
)

// This is a public ingestion token, not a personal or project secret API key.
const projectToken = "phc_kBqNf6Fx4wEaDzDthoJBqvbVVRCVZZ5Pk8CbeZK49RBp"
const apiHost = "https://us.i.posthog.com"

const uploadTimeout = 2500 * time.Millisecond
const shutdownTimeout = 3 * time.Second

type eventClient interface {
	Enqueue(posthog.Message) error
	Close() error
}

type Tracker struct {
	client eventClient
	id     string
}

// New is deliberately lazy: --no-analytics does not initialize the SDK or
// create an installation ID. Tests constructing CLI commands also stay offline.
func New() *Tracker { return &Tracker{} }

func disabled() bool {
	for _, key := range []string{"CYEAM_CLI_NO_ANALYTICS", "DO_NOT_TRACK", "CI"} {
		switch strings.ToLower(os.Getenv(key)) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}

func clientConfig() posthog.Config {
	return posthog.Config{
		Endpoint:           apiHost,
		DisableGeoIP:       posthog.Ptr(false), // The request comes from the CLI user's machine.
		IsServer:           posthog.Ptr(false),
		BatchSize:          1,
		MaxRetries:         posthog.Ptr(0),
		BatchUploadTimeout: uploadTimeout,
		ShutdownTimeout:    shutdownTimeout,
		Logger:             posthog.StdLogger(log.New(io.Discard, "", 0), false),
	}
}

// CommandStarted receives Cobra's canonical path, not os.Args. PostHog's SDK
// adds $os, $os_version, $os_distro (Linux), and $go_version automatically.
func (t *Tracker) CommandStarted(path string, info version.Info) {
	if disabled() {
		return
	}
	if t.client == nil {
		client, err := posthog.NewWithConfig(projectToken, clientConfig())
		if err != nil {
			return
		}
		t.client = client
		home, err := os.UserHomeDir()
		if err == nil {
			t.id = installationID(filepath.Join(home, ".cyeam"))
		} else {
			t.id = randomID()
		}
	}
	parts := strings.Fields(path)
	command, subcommand := "", ""
	if len(parts) > 1 {
		command = parts[1]
	}
	if len(parts) > 2 {
		subcommand = strings.Join(parts[2:], " ")
	}
	_ = t.client.Enqueue(posthog.Capture{
		DistinctId: t.id,
		Event:      "cli_command_started",
		Properties: posthog.NewProperties().
			Set("app", "cyeam-cli").
			Set("command_path", path).
			Set("command", command).
			Set("subcommand", subcommand).
			Set("cli_version", info.Version).
			Set("os", info.GOOS).
			Set("architecture", info.GOARCH).
			Set("$process_person_profile", false),
	})
}

// Close flushes on success and failure, with a deadline for offline clients.
// Analytics errors never alter CLI output or exit status.
func (t *Tracker) Close() {
	if t.client != nil {
		_ = t.client.Close()
	}
}

func randomID() string {
	id := uuid.New()
	return hex.EncodeToString(id[:])
}

func validID(data []byte) bool {
	if len(data) != 32 {
		return false
	}
	_, err := hex.DecodeString(string(data))
	return err == nil
}

func installationID(dir string) string {
	path := filepath.Join(dir, "analytics-id")
	if data, err := os.ReadFile(path); err == nil && validID(data) {
		return string(data)
	}
	id := randomID()
	if os.MkdirAll(dir, 0700) != nil {
		return id
	}
	// Publish a complete file atomically. Concurrent CLI invocations cannot
	// overwrite one another's ID or observe a partially written file.
	file, err := os.CreateTemp(dir, ".analytics-id-*")
	if err != nil {
		return id
	}
	defer os.Remove(file.Name())
	_, writeErr := io.WriteString(file, id)
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		_ = os.Link(file.Name(), path)
		if data, err := os.ReadFile(path); err == nil && validID(data) {
			return string(data)
		}
	}
	// Read-only homes or unsupported filesystems use an ephemeral random ID.
	return id
}
