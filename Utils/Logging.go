package Utils

import (
	"fmt"
	"os"
	"time"

	"github.com/elucid503/Sprout-API-Go/Logs"
)

// Logging provides structured logging functionality.
type Logging struct{}

// getTimestamp returns the current timestamp formatted for log output.
func (L *Logging) getTimestamp() string {

	return time.Now().Format("2006-01-02 15:04:05");

}

// Info logs an informational message.
func (L *Logging) Info(Title string, Message string) {

	go Logs.Log(os.Getenv("SERVICE_ID"), Logs.LogLevelInfo, Title, Message);
	fmt.Printf("[INFO] [%s] %s: %s\n", L.getTimestamp(), Title, Message);

}

// Warn logs a warning message.
func (L *Logging) Warn(Title string, Message string) {

	go Logs.Log(os.Getenv("SERVICE_ID"), Logs.LogLevelWarning, Title, Message);
	fmt.Printf("[WARN] [%s] %s: %s\n", L.getTimestamp(), Title, Message);

}

// Error logs an error message.
func (L *Logging) Error(Title string, Message string) {

	go Logs.Log(os.Getenv("SERVICE_ID"), Logs.LogLevelError, Title, Message);
	fmt.Printf("[ERROR] [%s] %s: %s\n", L.getTimestamp(), Title, Message);

}

// Logger is the global logging instance.
var Logger = &Logging{};