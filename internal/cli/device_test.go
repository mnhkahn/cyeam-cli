package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/mnhkahn/cyeam-cli/internal/gateway"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeviceCommandsUseOwnedGatewayAPI(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	var tool string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/auth/config" {
			w.Write([]byte(`{"client_id":"e1f582c1-1568-4347-8c5b-1906164e637f","audience":"11111111-1111-1111-1111-111111111111","scope":"api://11111111-1111-1111-1111-111111111111/Device.Control"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer owned-user-token" {
			t.Error("missing per-user token")
			w.WriteHeader(401)
			return
		}
		switch {
		case r.URL.Path == "/v1/devices/tablet" && r.Method == "DELETE":
			w.Write([]byte(`{"revoked":true}`))
		case r.URL.Path == "/v1/devices":
			w.Write([]byte(`{"devices":[{"id":"tablet","online":true}]}`))
		case r.URL.Path == "/v1/devices/tablet/call":
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("missing idempotency key")
			}
			var v struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			json.NewDecoder(r.Body).Decode(&v)
			if v.Name == "self.camera.snapshot" && v.Arguments["resolution"] != "svga" {
				t.Errorf("wrong photo arguments: %#v", v.Arguments)
			}
			tool = v.Name
			calls++
			w.WriteHeader(202)
			w.Write([]byte(`{"operation_id":"` + strings.Repeat("a", 64) + `"}`))
		case strings.HasPrefix(r.URL.Path, "/v1/operations/"):
			if tool == "self.screen.screenshot" || tool == "self.camera.snapshot" {
				w.Write([]byte(`{"status":"succeeded","result":{"preview_url":"/tablet/v1/media/11111111-1111-1111-1111-111111111111.jpg"}}`))
			} else if tool == "self.homework.get_status" {
				w.Write([]byte(`{"status":"succeeded","result":{"content":[{"type":"text","text":"{\"timer_running\":true,\"current_task_id\":\"work\",\"elapsed_seconds\":123,\"tasks\":[]}"}]}}`))
			} else {
				w.Write([]byte(`{"status":"succeeded","result":{"content":[{"type":"text","text":"{\"completed\":true}"}]}}`))
			}
		case strings.Contains(r.URL.Path, "/media/"):
			w.Write([]byte{255, 216, 255, 0})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	deps := Dependencies{Gateway: func() (*gateway.Client, error) {
		c, e := gateway.NewAt(server.URL)
		if e == nil {
			c.Token = func(context.Context, string) (string, error) { return "owned-user-token", nil }
		}
		return c, e
	}}
	run := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		deps.Stdout = &out
		root := NewRootCommand(deps)
		root.SetArgs(append([]string{"--no-analytics", "device"}, args...))
		if e := root.Execute(); e != nil {
			t.Fatal(e)
		}
		return out.String()
	}
	if !strings.Contains(run("list"), `"ok":true`) {
		t.Fatal("missing JSON envelope")
	}
	path := filepath.Join(t.TempDir(), "screen.jpg")
	run("screenshot", "--out", path)
	image, e := os.ReadFile(path)
	if e != nil || len(image) != 4 {
		t.Fatalf("download failed: %v", e)
	}
	photo := filepath.Join(t.TempDir(), "photo.jpg")
	run("photo", "--resolution", "svga", "--out", photo)
	image, e = os.ReadFile(photo)
	if e != nil || !bytes.Equal(image, []byte{255, 216, 255, 0}) {
		t.Fatalf("photo download failed: %v", e)
	}
	status := run("homework-status")
	if !strings.Contains(status, `"timer_running":true`) || !strings.Contains(status, `"elapsed_seconds":123`) || strings.Contains(status, `"tasks"`) {
		t.Fatalf("wrong homework status: %s", status)
	}
	before := calls
	for _, args := range [][]string{{"photo", "--resolution", "invalid"}, {"photo", "--out", photo}} {
		root := NewRootCommand(deps)
		root.SetArgs(append([]string{"--no-analytics", "device"}, args...))
		if e := root.Execute(); e == nil {
			t.Fatal("accepted invalid capture request")
		}
	}
	if calls != before {
		t.Fatal("dispatched camera for rejected request")
	}
	run("speak", "--text", "休息一下")
	if !strings.Contains(run("unpair", "--device", "tablet"), `"revoked":true`) {
		t.Fatal("device revocation failed")
	}
	if tool != "self.audio.speak" || calls != 4 {
		t.Fatalf("wrong tools/calls: %s %d", tool, calls)
	}
}
func TestScreenshotDoesNotOverwriteOrFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	_ = os.WriteFile(target, []byte("original"), 0600)
	if e := saveScreenshot(target, []byte("changed"), false); e == nil {
		t.Fatal("overwrote without --force")
	}
	link := filepath.Join(dir, "link")
	if e := os.Symlink(target, link); e != nil {
		t.Skip(e)
	}
	if e := saveScreenshot(link, []byte("changed"), true); e == nil {
		t.Fatal("followed symlink")
	}
	value, _ := os.ReadFile(target)
	if string(value) != "original" {
		t.Fatal("modified target")
	}
}
