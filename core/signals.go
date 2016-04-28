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

// ReapChildren cleans up zombies
// - on SIGCHLD send wait4() (ref http://linux.die.net/man/2/waitpid)
func reapChildren() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGCHLD)
	go func() {
		// wait for signals on the channel until it closes
		for _ = range sig {
			for {
				// only 1 SIGCHLD can be handled at a time from the channel,
				// so we need to allow for the possibility that multiple child
				// processes have terminated while one is already being reaped.
				var wstatus syscall.WaitStatus
				if _, err := syscall.Wait4(-1, &wstatus,
					syscall.WNOHANG|syscall.WUNTRACED|syscall.WCONTINUED,
					nil); err == syscall.EINTR {
					continue
				}
				// return to the outer loop and wait for another signal
				break
			}
		}
	}()
}
