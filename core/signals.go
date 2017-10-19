package core

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals listens for and captures signals used for orchestration
func (a *App) handleSignals() {
	recvSig := make(chan os.Signal, 1)
	signal.Notify(recvSig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			sig := <-recvSig
			switch sig {
			case syscall.SIGINT:
				a.Terminate()
				return
			case syscall.SIGTERM:
				a.Terminate()
				return
			default:
			}
		}
	}()
}
