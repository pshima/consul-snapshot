package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
)

const (
	version = "0.2.4"
)

func main() {
	os.Exit(realMain())
}

// Just our main function to kick things off in a loop.
func realMain() int {

	args := os.Args[1:]
	for _, arg := range args {
		if arg == "-v" || arg == "-version" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}
	}

	cli := &cli.CLI{
		Args:     args,
		Commands: Commands,
		HelpFunc: cli.FilteredHelpFunc(
			CommandsInclude, cli.BasicHelpFunc("consul-snapshot")),
		HelpWriter: os.Stdout,
	}

	exitCode, err := cli.Run()
	if err != nil {
		log.Fatalf("Error executing CLI: %s", err.Error())
		return 1
	}

	return exitCode

}

func formattedVersion() string {
	return version
}
