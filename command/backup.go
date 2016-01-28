package command

import (
	"fmt"
	"github.com/pshima/consul-snapshot/backup"
)

type BackupCommand struct {
	Meta
	Version string
}

func (c *BackupCommand) Run(args []string) int {
	c.Ui.Info(fmt.Sprintf("v%v: Starting Consul Snapshot", c.Version))
	response := backup.BackupRunner("constant")
	// Actually need to return the proper response here.
	return response
}

func (c *BackupCommand) Synopsis() string {
	return "Starts a backup"
}

func (c *BackupCommand) Help() string {
	return `
Usage: consul-snapshot backup

Starts a backup.
`
}
