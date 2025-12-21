package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

const Ver = "v1.2.2"

// Paths holds all application directory paths
type Paths struct {
	Home     string // ~/.sv
	Root     string // ~/.sv/go (symlink to current version)
	Bin      string // ~/.sv/bin
	Cache    string // ~/.sv/cache (installed versions)
	Download string // ~/.sv/downloads
}

var paths *Paths

// initPaths initializes and validates all required directories
func initPaths() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	paths = &Paths{
		Home:     filepath.Join(homeDir, ".sv"),
		Root:     filepath.Join(homeDir, ".sv", "go"),
		Bin:      filepath.Join(homeDir, ".sv", "bin"),
		Cache:    filepath.Join(homeDir, ".sv", "cache"),
		Download: filepath.Join(homeDir, ".sv", "downloads"),
	}

	// Create required directories
	dirs := []string{paths.Download, paths.Cache, paths.Bin}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func main() {
	SetLogLevel("debug")
	app := cli.NewApp()
	app.Usage = "switch version"
	app.Version = Ver
	app.Compiled = time.Now()
	app.EnableBashCompletion = true
	//app.CustomAppHelpTemplate = "add sv to your ~/.bashrc or ~/.zshrc. export PATH=\"$HOME/.sv/bin:$PATH\""
	//app.Action = baseCmd
	app.Commands = []*cli.Command{
		{
			Name:      "list",
			Usage:     "show all local versions",
			UsageText: "sv ls",
			Action:    baseCmd,
			Aliases:   []string{"ls", "l"},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "remote",
					Aliases:  []string{"r"},
					Usage:    "show all remote versions",
					Required: false,
				},
			},
		}, {
			Name:      "use",
			Usage:     "input a specific local version",
			UsageText: "sv use <version>",
			Action:    baseCmd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "remote",
					Aliases:  []string{"r"},
					Usage:    "input a specific remote version",
					Required: false,
				},
			},
		}, {
			Name:      "install",
			Usage:     "install a specific remote version",
			UsageText: "sv install <version>",
			Action:    baseCmd,
			Aliases:   []string{"i"},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "latest",
					Usage:    "install the latest version",
					Required: false,
				},
			},
		}, {
			Name:      "uninstall",
			Usage:     "uninstall a specific local version",
			UsageText: "sv uninstall <version>",
			Action:    baseCmd,
			Aliases:   []string{"ui"},
		}, {
			Name:      "prune",
			Usage:     "remove old Go versions, keeping the most recent ones",
			UsageText: "sv prune [--keep N] [--all] [--dry-run]",
			Action:    baseCmd,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "keep",
					Aliases: []string{"k"},
					Usage:   "number of versions to keep (default: 2)",
					Value:   2,
				},
				&cli.BoolFlag{
					Name:    "all",
					Aliases: []string{"a"},
					Usage:   "remove all versions except current",
				},
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "show what would be deleted without actually deleting",
				},
			},
		}, {
			Name:      "current",
			Usage:     "show the currently active Go version",
			UsageText: "sv current",
			Action:    baseCmd,
			Aliases:   []string{"c"},
		}, {
			Name:      "where",
			Usage:     "show the installation path of a Go version",
			UsageText: "sv where <version>",
			Action:    baseCmd,
		}, {
			Name:      "latest",
			Usage:     "show the latest available Go version",
			UsageText: "sv latest",
			Action:    baseCmd,
		}, {
			Name:      "outdated",
			Usage:     "check if installed versions are outdated",
			UsageText: "sv outdated",
			Action:    baseCmd,
		}, {
			Name:      "upgrade",
			Usage:     "upgrade sv",
			UsageText: "sv upgrade",
			Action:    baseCmd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "force",
					Aliases:  []string{"f"},
					Usage:    "force upgrade",
					Required: false,
				},
			},
		},
	}
	app.Before = func(context *cli.Context) error {
		return initPaths()
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func baseCmd(c *cli.Context) (err error) {
	if err := runApp(c); err != nil {
		return cli.Exit(err, 0)
	}
	return
}

func runApp(c *cli.Context) (err error) {
	opts := &startOpts{
		cmd:    c.Command.Name,
		target: c.Args().First(),
		remote: c.Bool("remote"),
		force:  c.Bool("force"),
		latest: c.String("latest"),
		keep:   c.Int("keep"),
		all:    c.Bool("all"),
		dryRun: c.Bool("dry-run"),
	}
	a := newApp(opts)
	err = a.Start()
	if err != nil {
		return err
	}
	return
}
