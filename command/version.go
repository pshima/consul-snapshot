package command

import (
	"github.com/mitchellh/cli"
)

var _ cli.Command = (*VersionCommand)(nil)

// VersionCommand for getting the version
type VersionCommand struct {
	Meta

	Version string
}

// Help for the command
func (c *VersionCommand) Help() string {
	return ""
}

// Run is the runner for the command
func (c *VersionCommand) Run(args []string) int {
	c.UI.Output(c.Version)
	return 0
}

// Synopsis for the command
func (c *VersionCommand) Synopsis() string {
	return "Prints the version"
}
