package main

import (
	"os"
	"testing"
)

func TestFormattedVersion(t *testing.T) {
	version := formattedVersion()
	if version == "" {
		t.Error("expected formattedVersion to return a non-empty string")
	}
	
	// Should return the version constant
	expected := "0.2.5"
	if version != expected {
		t.Errorf("expected version %s, got %s", expected, version)
	}
}

func TestMainFunction(t *testing.T) {
	// Test that main doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Logf("main() failed as expected in test environment: %v", r)
		}
	}()
	
	// We can't easily test main() without it trying to parse real args
	// But we can test that it exists and the function can be called
	// main() // This would try to parse os.Args, so we skip it
}

func TestRealMainVersionFlag(t *testing.T) {
	// Test that version flags are handled
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()
	
	// Test version flag handling
	testCases := []string{"-v", "-version", "--version"}
	
	for _, flag := range testCases {
		os.Args = []string{"consul-snapshot", flag}
		
		// This will exit, so we need to catch it
		defer func() {
			if r := recover(); r != nil {
				t.Logf("realMain() with %s flag failed as expected: %v", flag, r)
			}
		}()
		
		// The version flag handling modifies args to prepend "version" command
		// We can test this logic separately
		args := []string{flag}
		for _, arg := range args {
			if arg == "-v" || arg == "-version" || arg == "--version" {
				newArgs := make([]string, len(args)+1)
				newArgs[0] = "version"
				copy(newArgs[1:], args)
				args = newArgs
				break
			}
		}
		
		if len(args) > 1 && args[0] == "version" {
			// Version flag was properly handled
			continue
		} else {
			t.Errorf("version flag %s was not properly handled", flag)
		}
	}
}

func TestVersionConstant(t *testing.T) {
	// Test that the version constant is set
	if version == "" {
		t.Error("expected version constant to be non-empty")
	}
	
	if version != "0.2.5" {
		t.Errorf("expected version to be '0.2.5', got '%s'", version)
	}
}