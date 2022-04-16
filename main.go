package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

var Version = "v1.0.0"

func main() {
	app := cli.NewApp()
	app.Usage = "switch version"
	app.Version = Version
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:    "list",
			Usage:   "show all versions",
			Action:  List,
			Aliases: []string{"l"},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}
}

func List(c *cli.Context) error {
	return nil
}
