package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals listens for and captures signals used for orchestration
func (a *App) handleSignals(ctx context.Context, cancel context.CancelFunc) {
	recvSig := make(chan os.Signal, 1)
	signal.Notify(recvSig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			select {
			case sig := <-recvSig:
				switch sig {
				case syscall.SIGINT:
					a.Terminate()
					return
				case syscall.SIGTERM:
					a.Terminate()
					return
				default:
				}
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
}
