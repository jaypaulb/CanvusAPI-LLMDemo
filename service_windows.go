//go:build windows

// Package main provides Windows service support for CanvusLocalLLM.
//
// service_windows.go implements the Windows Service interface using github.com/kardianos/service.
// This allows the application to run as a Windows background service with proper Start/Stop handling.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kardianos/service"
)

// Program implements service.Interface for Windows Service integration.
// It wraps the main application logic and provides Start/Stop lifecycle methods.
type Program struct {
	// ctx is the context used to signal shutdown
	ctx context.Context
	// cancel is the function to trigger shutdown
	cancel context.CancelFunc
	// exit is the channel to signal service exit
	exit chan struct{}
}

// Start is called when the service is started.
// It initializes the application and begins processing in a goroutine.
func (p *Program) Start(s service.Service) error {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.exit = make(chan struct{})

	// Start the main application logic in a goroutine
	go p.run()

	return nil
}

// Stop is called when the service is stopped.
// It signals the application to shut down gracefully.
func (p *Program) Stop(s service.Service) error {
	// Signal shutdown
	p.cancel()

	// Wait for clean shutdown with timeout
	select {
	case <-p.exit:
		// Clean shutdown completed
	case <-time.After(30 * time.Second):
		// Timeout waiting for shutdown
		return fmt.Errorf("timeout waiting for service to stop")
	}

	return nil
}

// run contains the main service logic.
// This is called from Start and runs until Stop is signaled.
func (p *Program) run() {
	defer close(p.exit)

	// The actual application logic would be initialized here.
	// For now, we delegate to the main application's run function.
	//
	// In a full implementation, this would:
	// 1. Load configuration
	// 2. Initialize the Monitor
	// 3. Start processing canvas updates
	// 4. Wait for shutdown signal
	//
	// The main() function in main.go handles this when not running as a service.
	// When running as a service, we need to integrate with the service lifecycle.

	// Wait for shutdown signal
	<-p.ctx.Done()
}

// ServiceConfig returns the service configuration for Windows.
func ServiceConfig() *service.Config {
	return &service.Config{
		Name:        "CanvusLocalLLM",
		DisplayName: "Canvus Local LLM Service",
		Description: "Integrates Canvus collaborative workspaces with local AI services via llama.cpp",
		Option: service.KeyValue{
			"StartType": "automatic",
		},
	}
}

// RunAsService runs the application as a Windows service.
// Returns true if running as a service, false if running interactively.
func RunAsService() (bool, error) {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return false, fmt.Errorf("failed to create service: %w", err)
	}

	// Check if we're running interactively
	if service.Interactive() {
		return false, nil
	}

	// Run as service
	err = s.Run()
	if err != nil {
		return true, fmt.Errorf("service run failed: %w", err)
	}

	return true, nil
}

// InstallService installs the application as a Windows service.
func InstallService() error {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Install()
	if err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("Service installed successfully")
	return nil
}

// UninstallService removes the Windows service.
func UninstallService() error {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Uninstall()
	if err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("Service uninstalled successfully")
	return nil
}

// StartService starts the Windows service.
func StartService() error {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("Service started successfully")
	return nil
}

// StopService stops the Windows service.
func StopService() error {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("Service stopped successfully")
	return nil
}

// RestartService stops and then starts the Windows service.
func RestartService() error {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	err = s.Restart()
	if err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	fmt.Println("Service restarted successfully")
	return nil
}

// ServiceStatus returns the current status of the Windows service.
func ServiceStatus() (service.Status, error) {
	prg := &Program{}
	svcConfig := ServiceConfig()

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("failed to create service: %w", err)
	}

	status, err := s.Status()
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("failed to get service status: %w", err)
	}

	return status, nil
}

// PrintServiceUsage prints the help/usage information for service commands.
func PrintServiceUsage() {
	fmt.Println("CanvusLocalLLM Service Management")
	fmt.Println()
	fmt.Println("Usage: CanvusLocalLLM.exe <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install    Install the application as a Windows service")
	fmt.Println("  uninstall  Remove the Windows service (alias: remove)")
	fmt.Println("  start      Start the Windows service")
	fmt.Println("  stop       Stop the Windows service")
	fmt.Println("  restart    Restart the Windows service (stop then start)")
	fmt.Println("  status     Show the current service status")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Run without arguments to start the application in foreground mode.")
}

// ServiceMain is the entry point for service management commands.
// It processes the command-line arguments and dispatches to the appropriate
// service function. Returns true if a service command was handled, false otherwise.
// This is the main entry point that should be called from main() before
// starting the normal application.
func ServiceMain(args []string) bool {
	return HandleServiceCommand(args)
}

// HandleServiceCommand handles service-related command-line arguments.
// Returns true if a service command was handled, false otherwise.
func HandleServiceCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	var err error
	switch args[1] {
	case "install":
		err = InstallService()
	case "uninstall", "remove":
		err = UninstallService()
	case "start":
		err = StartService()
	case "stop":
		err = StopService()
	case "restart":
		err = RestartService()
	case "status":
		status, statusErr := ServiceStatus()
		if statusErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", statusErr)
			os.Exit(1)
		}
		switch status {
		case service.StatusRunning:
			fmt.Println("Service is running")
		case service.StatusStopped:
			fmt.Println("Service is stopped")
		default:
			fmt.Println("Service status unknown")
		}
		return true
	case "help", "-h", "--help", "-help":
		PrintServiceUsage()
		return true
	default:
		return false
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return true
}
