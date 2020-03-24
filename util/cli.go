package util

import (
	"github.com/urfave/cli"
)

func Die(c *cli.Context, msg string) {
	cli.ShowSubcommandHelp(c)
	Fatal(msg)
}
