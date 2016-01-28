package main

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/pshima/consul-snapshot/command"
)

var (
	// Commands holds the command definition
	Commands map[string]cli.CommandFactory
	// CommandsInclude holds the list of available commands
	CommandsInclude []string
)

// UI for commands
var UI cli.Ui

const (
	//ErrorPrefix is the string used to prefix error messages
	ErrorPrefix = "[ERR] "
	//OutputPrefix is the string used to prefix regular output
	OutputPrefix = "[INFO] "
)

func init() {
	UI = &cli.ColoredUi{
		OutputColor: cli.UiColorNone,
		InfoColor:   cli.UiColorNone,
		ErrorColor:  cli.UiColorRed,
		WarnColor:   cli.UiColorYellow,
		Ui: &cli.PrefixedUi{
			AskPrefix:    OutputPrefix,
			OutputPrefix: OutputPrefix,
			InfoPrefix:   OutputPrefix,
			ErrorPrefix:  ErrorPrefix,
			Ui:           &cli.BasicUi{Writer: os.Stdout},
		},
	}

	meta := command.Meta{
		UI: UI,
	}

	CommandsInclude = []string{
		"backup",
		"restore",
		"version",
	}

	Commands = map[string]cli.CommandFactory{
		"backup": func() (cli.Command, error) {
			return &command.BackupCommand{
				Meta:    meta,
				Version: formattedVersion(),
			}, nil
		},

		"restore": func() (cli.Command, error) {
			return &command.RestoreCommand{
				Meta:    meta,
				Version: formattedVersion(),
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Meta:    meta,
				Version: formattedVersion(),
			}, nil
		},
	}
}
