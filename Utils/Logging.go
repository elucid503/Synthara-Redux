package Utils

import (
	"fmt"
	"time"
)

// Logging provides structured logging functionality.
type Logging struct{}

// getTimestamp returns the current timestamp formatted for log output.
func (L *Logging) getTimestamp() string {

	return time.Now().Format("2006-01-02 15:04:05");

}

// Info logs an informational message.
func (L *Logging) Info(Message string) {

	fmt.Printf("[INFO] [%s] %s\n", L.getTimestamp(), Message);

}

// Warn logs a warning message.
func (L *Logging) Warn(Message string) {

	fmt.Printf("[WARN] [%s] %s\n", L.getTimestamp(), Message);

}

// Error logs an error message.
func (L *Logging) Error(Message string) {

	fmt.Printf("[ERROR] [%s] %s\n", L.getTimestamp(), Message);

}

// Logger is the global logging instance.
var Logger = &Logging{};