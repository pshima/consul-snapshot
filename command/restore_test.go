package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestRestoreCommand_Synopsis(t *testing.T) {
	c := &RestoreCommand{}
	synopsis := c.Synopsis()
	expected := "Starts a Restore"
	if synopsis != expected {
		t.Errorf("expected synopsis %q, got %q", expected, synopsis)
	}
}

func TestRestoreCommand_Help(t *testing.T) {
	c := &RestoreCommand{}
	help := c.Help()
	if !strings.Contains(help, "Usage: consul-snapshot restore") {
		t.Error("expected help to contain usage information")
	}
	if !strings.Contains(help, "filename.backup") {
		t.Error("expected help to contain filename example")
	}
}

func TestRestoreCommand_Run_NoArgs(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}, ErrorWriter: &bytes.Buffer{}}
	c := &RestoreCommand{
		Meta:    Meta{UI: ui},
		Version: "test",
	}

	code := c.Run([]string{})
	if code != 1 {
		t.Errorf("expected exit code 1 for no args, got %d", code)
	}

	errorOutput := ui.ErrorWriter.(*bytes.Buffer).String()
	if !strings.Contains(errorOutput, "You need to specify a restore file path") {
		t.Error("expected error message about missing file path")
	}
}

func TestRestoreCommand_Run_TooManyArgs(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}, ErrorWriter: &bytes.Buffer{}}
	c := &RestoreCommand{
		Meta:    Meta{UI: ui},
		Version: "test",
	}

	code := c.Run([]string{"file1", "file2"})
	if code != 1 {
		t.Errorf("expected exit code 1 for too many args, got %d", code)
	}

	errorOutput := ui.ErrorWriter.(*bytes.Buffer).String()
	if !strings.Contains(errorOutput, "You need to specify a restore file path") {
		t.Error("expected error message about file path specification")
	}
}

func TestRestoreCommand_Run_WithValidArg(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}, ErrorWriter: &bytes.Buffer{}}
	c := &RestoreCommand{
		Meta:    Meta{UI: ui},
		Version: "test",
	}

	// This will fail because we don't have S3 credentials, but we can test the initial logic
	_ = c.Run([]string{"test-backup.tar.gz"})
	
	output := ui.Writer.(*bytes.Buffer).String()
	if !strings.Contains(output, "Starting Consul Snapshot") {
		t.Error("expected output to contain startup message")
	}
	
	// The command will likely fail due to missing S3 config, but that's expected in test
	// We're just testing the command structure, not the full restore process
}