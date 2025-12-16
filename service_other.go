//go:build !windows

// Package main provides stubs for service functions on non-Windows platforms.
package main

// RunAsService is a no-op on non-Windows platforms.
// Returns false to indicate the application should run normally.
func RunAsService() (bool, error) {
	return false, nil
}

// HandleServiceCommand is a no-op on non-Windows platforms.
// Returns false to indicate no service command was handled.
func HandleServiceCommand(args []string) bool {
	return false
}
