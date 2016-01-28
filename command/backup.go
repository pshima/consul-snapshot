package command

import (
	"fmt"
	"github.com/pshima/consul-snapshot/backup"
)

// BackupCommand for running backups
type BackupCommand struct {
	Meta
	Version string
}

// Run the backup via backup.Runner
func (c *BackupCommand) Run(args []string) int {
	c.UI.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	response := backup.Runner("constant")
	// Actually need to return the proper response here.
	return response
}

// Synopsis of the command
func (c *BackupCommand) Synopsis() string {
	return "Starts a backup"
}

// Help for the command
func (c *BackupCommand) Help() string {
	return `
Usage: consul-snapshot backup

Starts a backup that repeats over an interval.
`
}
