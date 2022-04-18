package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Usage = "switch version"
	app.Version = "v1.0.0"
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

func List(c *cli.Context) (err error) {
	if err := runApp(c); err != nil {
		return cli.Exit(err, 1)
	}
	return
}

func runApp(c *cli.Context) (err error) {
	a := newApp()
	err = a.Start()
	if err != nil {
		return err
	}
	return
}
