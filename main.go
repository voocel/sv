package main

import (
	"context"
	"errors"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

// version is stamped from the git tag at release time (GoReleaser / Makefile
// pass -ldflags "-X main.version=..."); "dev" marks an unstamped local build.
var version = "dev"

func main() {
	app, err := newApp()
	if err != nil {
		errorf("%v", err)
		os.Exit(1)
	}

	if err := rootCommand(app).Run(context.Background(), os.Args); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			os.Exit(130)
		}
		errorf("%v", err)
		os.Exit(1)
	}
}

func rootCommand(a *App) *cli.Command {
	return &cli.Command{
		Name:                  "sv",
		Usage:                 "a lightweight Go version manager",
		Version:               version,
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls", "l"},
				Usage:     "list installed versions and switch interactively",
				UsageText: "sv list [-r]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "remote", Aliases: []string{"r"}, Usage: "list remote versions"},
				},
				Action: a.cmdList,
			},
			{
				Name:      "use",
				Usage:     "switch to a specific Go version",
				UsageText: "sv use <version>",
				Action:    a.cmdUse,
			},
			{
				Name:      "install",
				Aliases:   []string{"i"},
				Usage:     "install a specific Go version",
				UsageText: "sv install <version> | sv install --latest",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "latest", Usage: "install the latest stable version"},
				},
				Action: a.cmdInstall,
			},
			{
				Name:      "uninstall",
				Aliases:   []string{"ui"},
				Usage:     "uninstall a specific Go version",
				UsageText: "sv uninstall <version>",
				Action:    a.cmdUninstall,
			},
			{
				Name:      "prune",
				Usage:     "remove old Go versions, keeping the most recent ones",
				UsageText: "sv prune [--keep N] [--all] [--dry-run]",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "keep", Aliases: []string{"k"}, Usage: "number of versions to keep", Value: 2},
					&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "remove all versions except the current one"},
					&cli.BoolFlag{Name: "dry-run", Usage: "show what would be removed without removing"},
				},
				Action: a.cmdPrune,
			},
			{
				Name:      "current",
				Aliases:   []string{"c"},
				Usage:     "show the currently active Go version",
				UsageText: "sv current",
				Action:    a.cmdCurrent,
			},
			{
				Name:      "where",
				Usage:     "show the installation path of a Go version",
				UsageText: "sv where [version]",
				Action:    a.cmdWhere,
			},
			{
				Name:      "latest",
				Usage:     "show the latest available Go version",
				UsageText: "sv latest",
				Action:    a.cmdLatest,
			},
			{
				Name:      "outdated",
				Usage:     "check whether installed versions are outdated",
				UsageText: "sv outdated",
				Action:    a.cmdOutdated,
			},
			{
				Name:  "self",
				Usage: "manage sv itself",
				Commands: []*cli.Command{
					{
						Name:      "upgrade",
						Usage:     "upgrade sv to the latest version",
						UsageText: "sv self upgrade [--force]",
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "reinstall even if up to date"},
						},
						Action: a.cmdSelfUpgrade,
					},
					{
						Name:      "uninstall",
						Usage:     "uninstall sv and all installed Go versions",
						UsageText: "sv self uninstall",
						Action:    a.cmdSelfUninstall,
					},
				},
			},
		},
	}
}
