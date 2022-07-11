package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	svHome     string
	svRoot     string
	svBin      string
	svCache    string
	svDownload string
)

const Ver = "v1.1.2"

func main() {
	l.SetLevel("debug")
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
		}, {
			Name:      "uninstall",
			Usage:     "uninstall a specific local version",
			UsageText: "sv uninstall <version>",
			Action:    baseCmd,
			Aliases:   []string{"ui"},
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
	app.Before = func(context *cli.Context) (err error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		svHome = filepath.Join(homeDir, ".sv")
		svRoot = filepath.Join(svHome, "go")
		svBin = filepath.Join(svHome, "bin")
		svDownload = filepath.Join(svHome, "downloads")
		svCache = filepath.Join(svHome, "cache")
		if err = os.MkdirAll(svDownload, 0755); err != nil {
			return err
		}
		if err = os.MkdirAll(svCache, 0755); err != nil {
			return err
		}
		if err = os.MkdirAll(svBin, 0755); err != nil {
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
	}
	a := newApp(opts)
	err = a.Start()
	if err != nil {
		return err
	}
	return
}
