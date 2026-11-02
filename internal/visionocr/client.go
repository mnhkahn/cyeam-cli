// Package visionocr runs Apple's on-device text recognition through the system
// JavaScript/Objective-C bridge. No compiler, downloaded model or CGO is needed.
package visionocr

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

//go:embed recognize.js
var script string

var ErrUnavailable = errors.New("Apple Vision is unavailable on this device")

type Alternative struct {
	Line paddleocr.Line `json:"line"`
	Kind string         `json:"kind"`
}
type Result struct {
	Lines        []paddleocr.Line `json:"lines"`
	Alternatives []Alternative    `json:"alternatives,omitempty"`
	RawLines     []paddleocr.Line `json:"raw_lines,omitempty"`
}

func Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := os.Stat("/usr/bin/osascript")
	return err == nil
}

func RecognizeRaw(ctx context.Context, image []byte) ([]byte, error) {
	if !Available() {
		return nil, ErrUnavailable
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", "-l", "JavaScript", "-e", script)
	cmd.Stdin = bytes.NewReader(image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		// Native diagnostics may contain image contents/paths; keep errors generic.
		return nil, fmt.Errorf("Apple Vision recognition failed")
	}
	if _, err := Parse(data); err != nil {
		return nil, err
	}
	return data, nil
}

func Parse(data []byte) (Result, error) {
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("invalid Apple Vision response")
	}
	if len(result.Lines) == 0 {
		return result, fmt.Errorf("Apple Vision returned no text")
	}
	return result, nil
}
