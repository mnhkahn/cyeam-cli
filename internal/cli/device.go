package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mnhkahn/cyeam-cli/internal/gateway"
	"github.com/spf13/cobra"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func gatewayClient(deps Dependencies) (*gateway.Client, error) {
	if deps.Gateway != nil {
		return deps.Gateway()
	}
	return gateway.New()
}
func newDeviceCommand(deps Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "device", Short: "Control your paired Homework devices through the independent gateway"}
	var selected string
	root.PersistentFlags().StringVar(&selected, "device", "", "device ID; required when more than one device is online")
	root.AddCommand(&cobra.Command{Use: "login", Short: "Authorize the gateway using your Microsoft account", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		return c.Login(cmd.Context(), os.Stdout, os.Stderr)
	}})
	for _, entry := range []struct{ name, path, description string }{{"list", "/v1/devices", "List your paired devices"}, {"pair", "/v1/pairings", "Generate a five-minute pairing QR payload for your account"}} {
		entry := entry
		root.AddCommand(&cobra.Command{Use: entry.name, Short: entry.description, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			c, e := gatewayClient(deps)
			if e != nil {
				return e
			}
			method := "GET"
			var value any
			if entry.name == "pair" {
				method = "POST"
				value = map[string]any{}
			}
			result, e := c.Do(cmd.Context(), method, entry.path, value, "")
			if e != nil {
				return e
			}
			_, e = fmt.Fprintln(cmd.OutOrStdout(), string(result))
			return e
		}})
	}
	root.AddCommand(&cobra.Command{Use: "tools", Short: "List tools declared by an online device", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		id, e := c.Device(cmd.Context(), selected)
		if e != nil {
			return e
		}
		raw, e := c.Do(cmd.Context(), "GET", "/v1/devices/"+id+"/tools", nil, "")
		if e != nil {
			return e
		}
		_, e = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return e
	}})
	root.AddCommand(&cobra.Command{Use: "operation <id>", Short: "Inspect an existing operation without executing it again", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(args[0]) {
			return fmt.Errorf("invalid operation ID")
		}
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		raw, e := c.Do(cmd.Context(), "GET", "/v1/operations/"+args[0], nil, "")
		if e != nil {
			return e
		}
		_, e = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return e
	}})
	root.AddCommand(&cobra.Command{Use: "unpair", Short: "Revoke an explicitly selected device binding", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if selected == "" {
			return fmt.Errorf("--device is required when revoking a device")
		}
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		id, e := c.Device(cmd.Context(), selected)
		if e != nil {
			return e
		}
		raw, e := c.Do(cmd.Context(), "DELETE", "/v1/devices/"+id, nil, "")
		if e != nil {
			return e
		}
		_, e = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return e
	}})
	var out string
	var force bool
	var screenshotKey string
	shot := &cobra.Command{Use: "screenshot", Short: "Capture the current display and download its JPEG", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		id, e := c.Device(cmd.Context(), selected)
		if e != nil {
			return e
		}
		// Check the output before dispatching a screenshot, not after it has been taken.
		if info, e := os.Lstat(out); e == nil && (info.Mode()&os.ModeSymlink != 0 || info.IsDir() || !force) {
			return fmt.Errorf("output already exists or is unsafe; choose another --out, or --force for a regular file")
		} else if e != nil && !os.IsNotExist(e) {
			return e
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()
		raw, e := c.Call(ctx, id, "self.screen.screenshot", map[string]any{}, screenshotKey)
		if e != nil {
			return e
		}
		var result struct {
			Preview string `json:"preview_url"`
		}
		if e = json.Unmarshal(raw, &result); e != nil {
			return e
		}
		if !regexp.MustCompile(`^/tablet/v1/media/[a-f0-9-]{36}\.jpg$`).MatchString(result.Preview) {
			return fmt.Errorf("device returned an invalid screenshot reference")
		}
		image, e := c.Do(ctx, "GET", "/v1/devices/"+id+"/media/"+filepath.Base(result.Preview), nil, "")
		if e != nil {
			return e
		}
		if len(image) < 3 || image[0] != 255 || image[1] != 216 || image[2] != 255 {
			return fmt.Errorf("gateway did not return a JPEG")
		}
		if e = saveScreenshot(out, image, force); e != nil {
			return e
		}
		absolute, _ := filepath.Abs(out)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"device_id": id, "path": absolute, "bytes": len(image)})
	}}
	shot.Flags().StringVar(&out, "out", "screenshot.jpg", "output JPEG path")
	shot.Flags().BoolVar(&force, "force", false, "replace an existing regular output file")
	shot.Flags().StringVar(&screenshotKey, "idempotency-key", "", "reuse this key when retrying an uncertain operation")
	root.AddCommand(shot)
	var text, language, idem string
	var rate float64
	speech := &cobra.Command{Use: "speak", Short: "Speak text on the device and wait for playback completion", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if strings.TrimSpace(text) == "" || len([]rune(text)) > 300 {
			return fmt.Errorf("--text must contain 1–300 characters")
		}
		if language != "zh-CN" && language != "en-US" {
			return fmt.Errorf("--language must be zh-CN or en-US")
		}
		if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0.5 || rate > 2 {
			return fmt.Errorf("--rate must be between 0.5 and 2")
		}
		c, e := gatewayClient(deps)
		if e != nil {
			return e
		}
		id, e := c.Device(cmd.Context(), selected)
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()
		raw, e := c.Call(ctx, id, "self.audio.speak", map[string]any{"text": text, "language": language, "rate": rate}, idem)
		if e != nil {
			return e
		}
		var result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if e = json.Unmarshal(raw, &result); e != nil {
			return e
		}
		for _, content := range result.Content {
			var completed map[string]any
			if json.Unmarshal([]byte(content.Text), &completed) == nil && completed["completed"] == true {
				completed["device_id"] = id
				return json.NewEncoder(cmd.OutOrStdout()).Encode(completed)
			}
		}
		return fmt.Errorf("device did not confirm speech playback completion")
	}}
	speech.Flags().StringVar(&text, "text", "", "text to speak (up to 300 characters)")
	_ = speech.MarkFlagRequired("text")
	speech.Flags().StringVar(&language, "language", "zh-CN", "zh-CN or en-US")
	speech.Flags().Float64Var(&rate, "rate", 1, "speech rate (0.5–2)")
	speech.Flags().StringVar(&idem, "idempotency-key", "", "reuse this key when retrying an uncertain operation")
	root.AddCommand(speech)
	return root
}
func saveScreenshot(path string, data []byte, force bool) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".cyeam-screen-*")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if force {
		if info, e := os.Lstat(path); e == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
			return fmt.Errorf("unsafe output path")
		}
		return os.Rename(temp, path)
	}
	return os.Link(temp, path)
}
