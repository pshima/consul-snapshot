package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestVersionCommand_Run(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}}
	c := &VersionCommand{
		Meta:    Meta{UI: ui},
		Version: "1.2.3",
	}

	code := c.Run([]string{})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	output := ui.Writer.(*bytes.Buffer).String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("expected version output to contain '1.2.3', got %q", output)
	}
}

func TestVersionCommand_Synopsis(t *testing.T) {
	c := &VersionCommand{}
	synopsis := c.Synopsis()
	expected := "Prints the version"
	if synopsis != expected {
		t.Errorf("expected synopsis %q, got %q", expected, synopsis)
	}
}

func TestVersionCommand_Help(t *testing.T) {
	c := &VersionCommand{}
	help := c.Help()
	if help != "" {
		t.Errorf("expected empty help, got %q", help)
	}
}