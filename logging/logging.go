package logging

import (
	"fmt"
	"log"
)

// LogHandler logs messages to both console and file with optional emoji
func LogHandler(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)

	// Log to console with emoji
	fmt.Println(message)

	// Log to file with timestamp and source info
	log.Print(message)
}
