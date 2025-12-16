package main

import (
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
