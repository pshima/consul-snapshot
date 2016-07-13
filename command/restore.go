package command

import (
	"fmt"
	"github.com/pshima/consul-snapshot/restore"
)

// RestoreCommand for running restores
type RestoreCommand struct {
	Meta
	Version string
}

// Run the restore through restore.Runner
func (c *RestoreCommand) Run(args []string) int {
	if len(args) != 1 {
		c.UI.Error("You need to specify a restore file path from base of bucket")
		return 1
	}

	c.UI.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	response := restore.Runner(args[0])
	return response
}

// Synopsis of the command
func (c *RestoreCommand) Synopsis() string {
	return "Starts a Restore"
}

// Help for the command
func (c *RestoreCommand) Help() string {
	return `
Usage: consul-snapshot restore filename.backup

Starts a restore process
`
}
