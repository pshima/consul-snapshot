package command

import (
	"flag"
	"fmt"

	"github.com/mitchellh/cli"
	"github.com/pshima/consul-snapshot/backup"
)

// BackupCommand for running backups
type BackupCommand struct {
	Meta
	Version string
}

// Run the backup via backup.Runner
func (c *BackupCommand) Run(args []string) int {
	// Set flags
	var flagOnce bool
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.BoolVar(&flagOnce, "once", false, "")
	// Parse flags
	if err := fs.Parse(args); err != nil {
		return cli.RunResultHelp
	}

	c.UI.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	response := backup.Runner(c.Version, flagOnce)
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

Options:
  -once           Run consul-snapshot only once, instead of running on an interval
`
}
