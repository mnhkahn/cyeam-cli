package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	LatestVersion string `json:"latest_version"`
	CheckedAt     int64  `json:"checked_at"`
}

const cacheTTL = 24 * time.Hour

func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cyeam")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func statePath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "update-state.json"), nil
}

func LoadState() (*State, error) {
	p, err := statePath()
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveState(s *State) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	body, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(p, body, 0644)
}

func (s *State) IsStale() bool {
	if s.CheckedAt == 0 {
		return true
	}
	return time.Since(time.Unix(s.CheckedAt, 0)) > cacheTTL
}

func ShouldCheck() bool {
	for _, key := range []string{"CI", "BUILD_NUMBER", "RUN_ID"} {
		if os.Getenv(key) != "" {
			return false
		}
	}
	if os.Getenv("CYEAM_CLI_NO_UPDATE_NOTIFIER") == "1" {
		return false
	}
	return true
}

func CheckCached(current string) (string, bool) {
	state, err := LoadState()
	if err != nil || state.LatestVersion == "" {
		return "", false
	}
	if state.LatestVersion != current && state.LatestVersion != "" {
		return state.LatestVersion, true
	}
	return "", false
}

func RefreshCache(current string) (string, bool) {
	state, err := LoadState()
	if err == nil && !state.IsStale() {
		if state.LatestVersion != "" && state.LatestVersion != current {
			return state.LatestVersion, true
		}
		return "", false
	}

	tag, err := FetchLatestTag()
	if err != nil {
		return "", false
	}

	_ = SaveState(&State{
		LatestVersion: tag,
		CheckedAt:     time.Now().Unix(),
	})

	if tag != current {
		return tag, true
	}
	return "", false
}

func FetchLatestTag() (string, error) {
	u := GitHubUpdater{Repo: "mnhkahn/cyeam-cli"}
	return u.latestTag(context.Background())
}