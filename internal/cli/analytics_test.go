package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mnhkahn/cyeam-cli/internal/version"
	"github.com/spf13/cobra"
)

func TestAnalyticsRecordsCanonicalSubcommandsWithoutArguments(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"nested", []string{"mo", "char", "detail", "private-input"}, "cyeam mo char detail"},
		{"alias", []string{"future", "s", "private-input"}, "cyeam future sub"},
		{"help", []string{"future", "sub", "--help"}, "cyeam future sub"},
		{"bare", []string{}, "cyeam"},
		{"opt-out", []string{"--no-analytics", "future", "sub", "private-input"}, ""},
		{"help-opt-out", []string{"future", "sub", "--help", "--no-analytics"}, ""},
		{"completion", []string{"completion", "bash"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			stdout := &bytes.Buffer{}
			root := NewRootCommand(Dependencies{
				Service: &fakeService{}, Stdout: stdout,
				CommandStarted: func(path string, info version.Info) {
					paths = append(paths, path)
					if info.Version == "" {
						t.Error("missing CLI version")
					}
				},
			})
			// Commands added later automatically inherit the same telemetry hook.
			parent := &cobra.Command{Use: "future"}
			parent.AddCommand(&cobra.Command{Use: "sub [text]", Aliases: []string{"s"}, Run: func(cmd *cobra.Command, args []string) {}})
			root.AddCommand(parent)
			root.SetArgs(tc.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(paths, ","); got != tc.want {
				t.Fatalf("tracked %q, want %q", got, tc.want)
			}
			if strings.Contains(strings.Join(paths, ","), "private-input") {
				t.Fatal("argument leaked")
			}
		})
	}
}

func TestAnalyticsRecordsCommandEvenWhenRunFails(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	var paths []string
	root := NewRootCommand(Dependencies{
		CommandStarted: func(path string, _ version.Info) { paths = append(paths, path) },
	})
	root.SetArgs([]string{"date", "holiday"}) // No Service: RunE fails after the start event.
	if err := root.Execute(); err == nil {
		t.Fatal("expected command failure")
	}
	if len(paths) != 1 || paths[0] != "cyeam date holiday" {
		t.Fatalf("events: %v", paths)
	}
}
