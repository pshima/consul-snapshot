package command

import (
	"fmt"

	"github.com/pshima/consul-snapshot/backup"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
)

// BackupCommand for running backups
type BackupCommand struct {
	Meta
	Version string
}

// Run the backup via backup.Runner
func (c *BackupCommand) Run(args []string) int {
	c.UI.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	conf := config.ParseConfig()
	client := &consul.Consul{Client: *consul.Client()}

	b := &backup.Backup{
		Config: conf,
		Client: client,
	}

	response := b.Runner("constant")
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
