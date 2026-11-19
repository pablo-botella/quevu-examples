package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// signalContext is done when the system asks the process to end.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
