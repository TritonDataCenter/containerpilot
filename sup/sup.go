// Package sup provides the child process reaper for PID1
package sup

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Run forks the ContainerPilot process and then starts signal handlers
// that will reap child processes and pass-thru SIGINT and SIGKILL to
// the ContainerPilot worker process.
func Run() {
	self, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Fatal("failed to find ContainerPilot binary: ", err)
	}
	proc, err := os.StartProcess(self, os.Args, &os.ProcAttr{Dir: "", Env: nil,
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}, Sys: nil})
	if err != nil {
		log.Fatal("failed to start ContainerPilot worker process:", err)
	}
	handleSignals(proc.Pid)
	proc.Wait()
}

// handleSignals listens for signals used to gracefully shutdown and
// passes them thru to the ContainerPilot worker process.
func handleSignals(pid int) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGCHLD, syscall.SIGUSR1)
	go func() {
		for signal := range sig {
			switch signal {
			case syscall.SIGINT:
				syscall.Kill(pid, syscall.SIGINT)
			case syscall.SIGTERM:
				syscall.Kill(pid, syscall.SIGTERM)
			case syscall.SIGUSR1:
				syscall.Kill(pid, syscall.SIGUSR1)
			case syscall.SIGCHLD:
				go reap()
			}
		}
	}()
}

// reaps child processes that have been reparented to PID1
func reap() {
	for {
	POLL:
		var wstatus syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &wstatus, 0, nil)
		switch err {
		case nil:
			if pid > 0 {
				goto POLL
			}
			return
		case syscall.ECHILD:
			return // no more children, we're done till next signal
		case syscall.EINTR:
			goto POLL
		default:
			return
		}
	}
}
