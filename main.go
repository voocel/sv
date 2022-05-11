package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

var (
	svHome     string
	svRoot     string
	svCache    string
	svDownload string
)

func main() {
	app := cli.NewApp()
	app.Usage = "switch version"
	app.Version = "v1.0.0"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "sv",
			Aliases:  []string{"s"},
			Usage:    "input a specific version",
			Required: false,
		}, &cli.StringFlag{
			Name:     "remote",
			Aliases:  []string{"r"},
			Usage:    "input a specific version",
			Required: false,
		},
	}
	app.Action = baseCmd
	app.Commands = []*cli.Command{
		{
			Name:    "list",
			Usage:   "show all local versions",
			Action:  baseCmd,
			Aliases: []string{"ls"},
		}, {
			Name:   "use",
			Usage:  "switch local versions",
			Action: baseCmd,
		},
	}
	app.Before = func(context *cli.Context) (err error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		svHome = filepath.Join(homeDir, "sv")
		svRoot = filepath.Join(svHome, "go")
		svDownload = filepath.Join(svHome, "downloads")
		svCache = filepath.Join(svHome, "cache")
		if err = os.MkdirAll(svDownload, 0755); err != nil {
			return err
		}
		if err = os.MkdirAll(svCache, 0755); err != nil {
			return err
		}
		return
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}
}

func baseCmd(c *cli.Context) (err error) {
	if err := runApp(c); err != nil {
		return cli.Exit(err, 1)
	}
	return
}

func runApp(c *cli.Context) (err error) {
	opts := &startOpts{
		cmd:    c.Command.Name,
		target: c.Args().First(),
	}
	a := newApp(opts)
	err = a.Start()
	if err != nil {
		return err
	}
	return
}
