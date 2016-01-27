package command

import (
	"github.com/mitchellh/cli"
)

var _ cli.Command = (*VersionCommand)(nil)

type VersionCommand struct {
	Meta

	Version string
}

func (c *VersionCommand) Help() string {
	return ""
}

func (c *VersionCommand) Run(args []string) int {
	c.Ui.Output(c.Version)
	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the version"
}
