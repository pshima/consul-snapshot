package command

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/cli"
)

func TestBackupCommand_Synopsis(t *testing.T) {
	c := &BackupCommand{}
	synopsis := c.Synopsis()
	expected := "Starts a backup"
	if synopsis != expected {
		t.Errorf("expected synopsis %q, got %q", expected, synopsis)
	}
}

func TestBackupCommand_Help(t *testing.T) {
	c := &BackupCommand{}
	help := c.Help()
	if !strings.Contains(help, "Usage: consul-snapshot backup") {
		t.Error("expected help to contain usage information")
	}
	if !strings.Contains(help, "-once") {
		t.Error("expected help to contain -once flag information")
	}
}

func TestBackupCommand_Run_InvalidArgs(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}, ErrorWriter: &bytes.Buffer{}}
	c := &BackupCommand{
		Meta:    Meta{UI: ui},
		Version: "test",
	}

	// Test with invalid flag
	code := c.Run([]string{"-invalid"})
	if code != cli.RunResultHelp {
		t.Errorf("expected exit code %d for invalid args, got %d", cli.RunResultHelp, code)
	}
}

func TestBackupCommand_Run_OutputMessage(t *testing.T) {
	ui := &cli.BasicUi{Writer: &bytes.Buffer{}, ErrorWriter: &bytes.Buffer{}}
	c := &BackupCommand{
		Meta:    Meta{UI: ui},
		Version: "test",
	}

	// Set required environment variables to prevent early exit
	os.Setenv("BACKUPINTERVAL", "60")
	os.Setenv("S3BUCKET", "test-bucket")
	os.Setenv("S3REGION", "us-east-1")
	defer func() {
		os.Unsetenv("BACKUPINTERVAL")
		os.Unsetenv("S3BUCKET")
		os.Unsetenv("S3REGION")
	}()

	// Just test the initial output before it tries to connect to consul
	// We'll catch the fatal exit and verify the startup message was printed
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Backup command failed as expected due to consul connection: %v", r)
		}
	}()

	// This will fail due to consul not being available, but we can check the initial output
	go func() {
		defer func() {
			recover() // Catch the log.Fatalf
		}()
		c.Run([]string{"-once"})
	}()

	// Give it a moment to print the startup message
	time.Sleep(100 * time.Millisecond)

	output := ui.Writer.(*bytes.Buffer).String()
	if !strings.Contains(output, "Starting Consul Snapshot") {
		t.Error("expected output to contain startup message")
	}
}