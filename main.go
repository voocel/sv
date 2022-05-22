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
	app.CustomAppHelpTemplate = "add sv to your ~/.bashrc or ~/.zshrc. export PATH=\"$HOME/.sv/bin:$PATH\""
	//app.Action = baseCmd
	app.Commands = []*cli.Command{
		{
			Name:    "list",
			Usage:   "show all local versions",
			Action:  baseCmd,
			Aliases: []string{"ls"},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "remote",
					Aliases:  []string{"r"},
					Usage:    "show all remote versions",
					Required: false,
				},
			},
		}, {
			Name:   "use",
			Usage:  "input a specific local version",
			Action: baseCmd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "remote",
					Aliases:  []string{"r"},
					Usage:    "input a specific remote version",
					Required: false,
				},
			},
		}, {
			Name:    "install",
			Usage:   "install a specific remote version",
			Action:  baseCmd,
			Aliases: []string{"i"},
		}, {
			Name:    "uninstall",
			Usage:   "uninstall a specific local version",
			Action:  baseCmd,
			Aliases: []string{"ui"},
		},
	}
	app.Before = func(context *cli.Context) (err error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		svHome = filepath.Join(homeDir, ".sv")
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
		log.Fatal(err)
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
		remote: c.Bool("remote"),
	}
	a := newApp(opts)
	err = a.Start()
	if err != nil {
		return err
	}
	return
}
