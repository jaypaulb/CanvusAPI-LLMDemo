package core

// Exit codes for the application.
// These follow Unix conventions where signal-based exits are 128 + signal number.
const (
	// ExitCodeSuccess indicates clean shutdown (exit code 0)
	ExitCodeSuccess = 0

	// ExitCodeError indicates an error occurred (exit code 1)
	ExitCodeError = 1

	// ExitCodeSIGINT indicates termination due to SIGINT (Ctrl+C)
	// Convention: 128 + 2 (SIGINT) = 130
	ExitCodeSIGINT = 130

	// ExitCodeSIGTERM indicates termination due to SIGTERM
	// Convention: 128 + 15 (SIGTERM) = 143
	ExitCodeSIGTERM = 143
)

// ExitCodeName returns a human-readable name for an exit code.
func ExitCodeName(code int) string {
	switch code {
	case ExitCodeSuccess:
		return "success"
	case ExitCodeError:
		return "error"
	case ExitCodeSIGINT:
		return "interrupted (SIGINT)"
	case ExitCodeSIGTERM:
		return "terminated (SIGTERM)"
	default:
		return "unknown"
	}
}

// IsSignalExit returns true if the exit code indicates a signal-based termination.
func IsSignalExit(code int) bool {
	return code == ExitCodeSIGINT || code == ExitCodeSIGTERM
}
