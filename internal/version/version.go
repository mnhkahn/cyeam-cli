package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GOOS      string
	GOARCH    string
}

func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

func (i Info) String() string {
	return fmt.Sprintf(
		"version: %s\ncommit: %s\nbuild_date: %s\ngoos: %s\ngoarch: %s\n",
		i.Version,
		i.Commit,
		i.BuildDate,
		i.GOOS,
		i.GOARCH,
	)
}
