package main

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/pshima/consul-snapshot/command"
)

var (
	Commands        map[string]cli.CommandFactory
	CommandsInclude []string
)

var Ui cli.Ui

const (
	ErrorPrefix  = "[ERR] "
	OutputPrefix = "[INFO] "
)

func init() {
	Ui = &cli.ColoredUi{
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
		Ui: Ui,
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
