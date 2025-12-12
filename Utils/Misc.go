package Utils

import (
	"os"
	"os/signal"
	"syscall"
)

func Hang() {

	SignalChan := make(chan os.Signal, 1)

	signal.Notify(SignalChan, syscall.SIGINT, syscall.SIGTERM) // Listens for termination signals

	<-SignalChan // Blocks until a signal is received
	
}