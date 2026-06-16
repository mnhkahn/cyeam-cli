package main

import (
	"os"

	"github.com/mnhkahn/cyeam-cli/internal/cli"
	"github.com/mnhkahn/cyeam-cli/internal/client"
	"github.com/mnhkahn/cyeam-cli/internal/cyeam"
	"github.com/mnhkahn/cyeam-cli/internal/update"
)

const baseURL = "https://www.cyeam.com"

func main() {
	httpClient := client.New(baseURL, nil)
	service := cyeam.NewService(httpClient, baseURL)
	updater := update.NewGitHubUpdater("mnhkahn/cyeam-cli", update.ArchiveInstaller{})
	cmd := cli.NewRootCommand(cli.Dependencies{
		Service: service,
		Stdout:  os.Stdout,
		Updater: updater,
	})
	if err := cmd.Execute(); err != nil {
		cli.WriteError(os.Stderr, err)
		os.Exit(1)
	}
}
