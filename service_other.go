//go:build !windows

// Package main provides stubs for service functions on non-Windows platforms.
//
// These stubs ensure the application compiles and runs on Linux and macOS,
// though service management commands are Windows-only features.
package main

import (
	"fmt"
)

// RunAsService is a no-op on non-Windows platforms.
// Returns false to indicate the application should run normally.
func RunAsService() (bool, error) {
	return false, nil
}

// ServiceMain is the entry point for service management commands.
// On non-Windows platforms, this is a no-op that returns false.
func ServiceMain(args []string) bool {
	return HandleServiceCommand(args)
}

// HandleServiceCommand is a no-op on non-Windows platforms.
// Returns false to indicate no service command was handled.
// On non-Windows systems, if a service command is explicitly passed,
// we print a helpful message indicating the limitation.
func HandleServiceCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check if user is trying to use a service command on non-Windows
	switch args[1] {
	case "install", "uninstall", "remove", "start", "stop", "restart", "status":
		fmt.Println("Service commands are only available on Windows.")
		fmt.Println("On Linux/macOS, use systemd, launchd, or run in foreground.")
		return true
	case "help", "-h", "--help", "-help":
		PrintServiceUsage()
		return true
	default:
		return false
	}
}

// PrintServiceUsage prints the help/usage information.
// On non-Windows platforms, indicates that service management is Windows-only.
func PrintServiceUsage() {
	fmt.Println("CanvusLocalLLM")
	fmt.Println()
	fmt.Println("Usage: canvuslocallm [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Note: Service management commands (install, uninstall, start, stop,")
	fmt.Println("      restart, status) are only available on Windows.")
	fmt.Println()
	fmt.Println("On Linux, consider using systemd to run as a service.")
	fmt.Println("On macOS, consider using launchd to run as a service.")
	fmt.Println()
	fmt.Println("Run without arguments to start the application in foreground mode.")
}
