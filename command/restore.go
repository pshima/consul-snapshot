package command

import (
	"fmt"
	"github.com/pshima/consul-snapshot/restore"
)

type RestoreCommand struct {
	Meta
	Version string
}

func (c *RestoreCommand) Run(args []string) int {
	if len(args) != 1 {
		c.Ui.Error("You need to specify a restore file path from base of bucket")
		return 1
	}

	c.Ui.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	response := restore.RestoreRunner(args[0])
	return response
}

func (c *RestoreCommand) Synopsis() string {
	return "Starts a Restore"
}

func (c *RestoreCommand) Help() string {
	return `
Usage: consul-snapshot restore filename.backup

Starts a restore process
`
}
