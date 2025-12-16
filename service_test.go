package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestHandleServiceCommand_NoArgs(t *testing.T) {
	// Empty args should return false
	handled := HandleServiceCommand([]string{})
	if handled {
		t.Error("HandleServiceCommand should return false for empty args")
	}
}

func TestHandleServiceCommand_SingleArg(t *testing.T) {
	// Single arg (just program name) should return false
	handled := HandleServiceCommand([]string{"program"})
	if handled {
		t.Error("HandleServiceCommand should return false for single arg")
	}
}

func TestHandleServiceCommand_UnknownCommand(t *testing.T) {
	// Unknown command should return false
	handled := HandleServiceCommand([]string{"program", "unknown"})
	if handled {
		t.Error("HandleServiceCommand should return false for unknown command")
	}
}

func TestHandleServiceCommand_Help(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"help", "help"},
		{"-h", "-h"},
		{"--help", "--help"},
		{"-help", "-help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			handled := HandleServiceCommand([]string{"program", tt.command})

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if !handled {
				t.Errorf("HandleServiceCommand should return true for %s command", tt.command)
			}

			// Verify help output contains key information
			if !strings.Contains(output, "CanvusLocalLLM") {
				t.Errorf("Help output should contain 'CanvusLocalLLM', got: %s", output)
			}
			if !strings.Contains(output, "help") {
				t.Errorf("Help output should contain 'help' command, got: %s", output)
			}
		})
	}
}

func TestHandleServiceCommand_ServiceCommands_NonWindows(t *testing.T) {
	// On non-Windows, service commands should return true but print a message
	// indicating they're Windows-only
	commands := []string{"install", "uninstall", "remove", "start", "stop", "restart", "status"}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			handled := HandleServiceCommand([]string{"program", cmd})

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if !handled {
				t.Errorf("HandleServiceCommand should return true for %s command on non-Windows", cmd)
			}

			// On non-Windows, should indicate the command is Windows-only
			if !strings.Contains(output, "Windows") {
				t.Errorf("Output should mention Windows, got: %s", output)
			}
		})
	}
}

func TestServiceMain(t *testing.T) {
	// ServiceMain should delegate to HandleServiceCommand

	// Test with no args
	handled := ServiceMain([]string{})
	if handled {
		t.Error("ServiceMain should return false for empty args")
	}

	// Test with single arg
	handled = ServiceMain([]string{"program"})
	if handled {
		t.Error("ServiceMain should return false for single arg")
	}

	// Test with unknown command
	handled = ServiceMain([]string{"program", "unknown"})
	if handled {
		t.Error("ServiceMain should return false for unknown command")
	}
}

func TestServiceMain_Help(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handled := ServiceMain([]string{"program", "help"})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	if !handled {
		t.Error("ServiceMain should return true for help command")
	}
}

func TestPrintServiceUsage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintServiceUsage()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains key information
	if !strings.Contains(output, "CanvusLocalLLM") {
		t.Errorf("PrintServiceUsage output should contain 'CanvusLocalLLM', got: %s", output)
	}
	if !strings.Contains(output, "help") {
		t.Errorf("PrintServiceUsage output should contain 'help', got: %s", output)
	}
}

func TestRunAsService_Interactive(t *testing.T) {
	// On non-Windows or in interactive mode, should return false
	isService, err := RunAsService()
	if err != nil {
		t.Errorf("RunAsService returned error: %v", err)
	}
	// In test environment, we're running interactively
	if isService {
		t.Error("RunAsService should return false in interactive/test mode")
	}
}
