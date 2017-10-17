// Package sup provides the child process reaper for PID1
package sup

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

// ConfigPath is input from the apps main function and includes the final/parsed
// path to the configuration file.
var ConfigPath string

// Run forks the ContainerPilot process and then starts signal handlers
// that will reap child processes and pass-thru SIGINT and SIGKILL to
// the ContainerPilot worker process.
func Run(configPath string) {
	ConfigPath = configPath
	self, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Fatal("failed to find ContainerPilot binary: ", err)
	}
	proc, err := os.StartProcess(self, os.Args, &os.ProcAttr{Dir: "", Env: nil,
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}, Sys: nil})
	if err != nil {
		log.Fatal("failed to start ContainerPilot worker process:", err)
	}
	passThroughSignals(proc.Pid)
	handleReaping(proc.Pid)
	proc.Wait()
}

// initClient loads the configuration so we can get the control socket and
// initializes the HTTPClient which callers will use for sending
// it commands
func initClient(configPath string) (*client.HTTPClient, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	httpclient, err := client.NewHTTPClient(cfg.Control.SocketPath)
	if err != nil {
		return nil, err
	}
	return httpclient, nil
}

// signalWorker POSTs a string representation of a signal to the PostSignal
// endpoint of the worker process control server.
func signalWorker(sig os.Signal) {
	var str string
	switch sig {
	case syscall.SIGHUP:
		str = "SIGHUP"
	case syscall.SIGUSR2:
		str = "SIGUSR2"
	}
	client, err := initClient(ConfigPath)
	if err != nil {
		log.Fatal("failed to init signal client:", err)
	}
	body, err := json.Marshal(map[string]string{"signal": str})
	if err != nil {
		log.Fatal("failed to marshal signal:", err)
	}
	if err = client.PostSignal(string(body)); err != nil {
		log.Fatal("failed to post signal:", err)
	}
}

// passThroughSignals listens for signals used to gracefully shutdown and
// passes them thru to the ContainerPilot worker process.
func passThroughSignals(pid int) {
	sigRecv := make(chan os.Signal, 1)
	signal.Notify(sigRecv,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)
	go func() {
		for sig := range sigRecv {
			switch sig {
			case syscall.SIGINT:
				syscall.Kill(pid, syscall.SIGINT)
			case syscall.SIGTERM:
				syscall.Kill(pid, syscall.SIGTERM)
			case syscall.SIGUSR1:
				syscall.Kill(pid, syscall.SIGUSR1)
			case syscall.SIGHUP, syscall.SIGUSR2:
				signalWorker(sig)
			}
		}
	}()
}

// handleReaping listens for the SIGCHLD signal only and triggers
// reaping of child processes
func handleReaping(pid int) {
	sigRecv := make(chan os.Signal, 1)
	signal.Notify(sigRecv, syscall.SIGCHLD)
	go func() {
		for {
			<-sigRecv
			reap()
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
