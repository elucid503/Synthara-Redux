package Utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func Hang() {

	SignalChan := make(chan os.Signal, 1)

	signal.Notify(SignalChan, syscall.SIGINT, syscall.SIGTERM) // Listens for termination signals

	<-SignalChan // Blocks until a signal is received

	Logger.Info("Shutting down and disconnecting Discord client...")

	DiscordClient.Close(context.TODO())
	
}