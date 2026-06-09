package version

import "testing"

func TestInfoStringIncludesBuildMetadata(t *testing.T) {
	info := Info{
		Version:   "v1.2.3",
		Commit:    "abc123",
		BuildDate: "2026-06-09T12:00:00Z",
		GOOS:      "darwin",
		GOARCH:    "arm64",
	}

	got := info.String()
	want := "version: v1.2.3\ncommit: abc123\nbuild_date: 2026-06-09T12:00:00Z\ngoos: darwin\ngoarch: arm64\n"
	if got != want {
		t.Fatalf("String() = %q", got)
	}
}
