package core

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals listens for and captures signals used for orchestration
func (a *App) handleSignals() {
	recvSig := make(chan os.Signal, 1)
	signal.Notify(recvSig,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGUSR2,
	)
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
			case syscall.SIGHUP, syscall.SIGUSR2:
				if s := toString(sig); s != "" {
					a.SignalEvent(s)
				}
			default:
			}
		}
	}()
}

func toString(sig os.Signal) string {
	switch sig {
	case syscall.SIGHUP:
		return "SIGHUP"
	case syscall.SIGUSR2:
		return "SIGUSR2"
	default:
		return ""
	}
}
