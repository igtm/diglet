package main

import (
	"fmt"
	"os"

	"github.com/buckhx/diglet/resources"
	"github.com/buckhx/diglet/tileserver"

	"github.com/codegangsta/cli"
)

//go:generate go run scripts/include.go

func main() {
	client(os.Args)
}

func die(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func client(args []string) {
	app := cli.NewApp()
	app.Name = "diglet"
	app.Usage = "Your friend in the tile business"
	app.Version = resources.Version
	app.Commands = []cli.Command{
		{
			Name:        "start",
			Usage:       "diglet start --mbtiles path/to/tiles",
			Description: "Starts the diglet server",
			Action: func(c *cli.Context) {
				p := c.String("port")
				mbt := c.String("mbtiles")
				if mbt == "" {
					cli.ShowSubcommandHelp(c)
					die("ERROR: --mbtiles flag is required")
				}
				server := tileserver.MBTServer(mbt, p)
				server.Run()
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "port",
					Value: "8080",
					Usage: "Port to bind",
				},
				cli.StringFlag{
					Name:  "mbtiles",
					Value: "",
					Usage: "REQUIRED: Path to mbtiles to serve",
				},
			},
		},
	}
	app.Run(args)
}
