package core

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals listens for and captures signals used for orchestration
func (a *App) handleSignals() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for signal := range sig {
			switch signal {
			case syscall.SIGUSR1:
				a.ToggleMaintenanceMode()
			case syscall.SIGTERM:
				a.Terminate()
			case syscall.SIGHUP:
				a.Reload()
			}
		}
	}()
}
