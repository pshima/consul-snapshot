package main

import (
	"testing"
	
	"github.com/mitchellh/cli"
)

func TestCommands(t *testing.T) {
	// Test that Commands map is properly initialized
	if Commands == nil {
		t.Fatal("Commands map should not be nil")
	}
	
	// Test that expected commands are present
	expectedCommands := []string{"backup", "restore", "version"}
	
	for _, cmd := range expectedCommands {
		if _, exists := Commands[cmd]; !exists {
			t.Errorf("expected command '%s' to be in Commands map", cmd)
		}
	}
}

func TestCommandsInclude(t *testing.T) {
	// Test that CommandsInclude slice is properly initialized
	if CommandsInclude == nil {
		t.Fatal("CommandsInclude should not be nil")
	}
	
	expectedCommands := []string{"backup", "restore", "version"}
	
	if len(CommandsInclude) != len(expectedCommands) {
		t.Errorf("expected %d commands in CommandsInclude, got %d", len(expectedCommands), len(CommandsInclude))
	}
	
	for _, expected := range expectedCommands {
		found := false
		for _, actual := range CommandsInclude {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command '%s' to be in CommandsInclude", expected)
		}
	}
}

func TestCommandFactories(t *testing.T) {
	// Test that command factories work
	for cmdName, factory := range Commands {
		cmd, err := factory()
		if err != nil {
			t.Errorf("command factory for '%s' returned error: %v", cmdName, err)
			continue
		}
		
		if cmd == nil {
			t.Errorf("command factory for '%s' returned nil command", cmdName)
			continue
		}
		
		// Test that the command implements the cli.Command interface
		if _, ok := cmd.(cli.Command); !ok {
			t.Errorf("command '%s' does not implement cli.Command interface", cmdName)
		}
		
		// Test basic command methods
		synopsis := cmd.Synopsis()
		if synopsis == "" && cmdName != "version" { // version command has empty synopsis
			t.Errorf("command '%s' has empty synopsis", cmdName)
		}
		
		help := cmd.Help()
		if help == "" && cmdName != "version" { // version command has empty help
			t.Errorf("command '%s' has empty help", cmdName)
		}
	}
}

func TestUI(t *testing.T) {
	// Test that UI is properly initialized
	if UI == nil {
		t.Fatal("UI should not be nil")
	}
	
	// Test that UI can be used
	UI.Output("test message")
	UI.Info("test info")
	UI.Warn("test warning")
	UI.Error("test error")
}

func TestConstants(t *testing.T) {
	// Test that constants are defined
	if ErrorPrefix == "" {
		t.Error("ErrorPrefix should not be empty")
	}
	
	if OutputPrefix == "" {
		t.Error("OutputPrefix should not be empty")
	}
	
	expectedErrorPrefix := "[ERR] "
	if ErrorPrefix != expectedErrorPrefix {
		t.Errorf("expected ErrorPrefix to be '%s', got '%s'", expectedErrorPrefix, ErrorPrefix)
	}
	
	expectedOutputPrefix := "[INFO] "
	if OutputPrefix != expectedOutputPrefix {
		t.Errorf("expected OutputPrefix to be '%s', got '%s'", expectedOutputPrefix, OutputPrefix)
	}
}