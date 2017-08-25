package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals listens for and captures signals used for orchestration
func (a *App) handleSignals(cancel context.CancelFunc) {
	recvSig := make(chan os.Signal, 1)
	signal.Notify(recvSig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		defer cancel()
		for sig := range recvSig {
			switch sig {
			case syscall.SIGINT:
				a.Terminate()
				return
			case syscall.SIGTERM:
				a.Terminate()
				return
			}
		}
	}()
}
