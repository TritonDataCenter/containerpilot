package containerbuddy

import (
	"flag"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"utils"

	log "github.com/Sirupsen/logrus"
)

// Main executes the containerbuddy CLI
func Main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	config, configErr := loadConfig()
	if configErr != nil {
		log.Fatal(configErr)
	}

	// Run the preStart handler, if any, and exit if it returns an error
	if preStartCode, err := run(config.preStartCmd); err != nil {
		os.Exit(preStartCode)
	}

	// Set up handlers for polling and to accept signal interrupts
	if 1 == os.Getpid() {
		reapChildren()
	}
	handleSignals(config)
	handlePolling(config)

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		config.Command = utils.ArgsToCmd(flag.Args())
		code, err := executeAndWait(config.Command)
		if err != nil {
			log.Errorln(err)
		}
		// Run the PostStop handler, if any, and exit if it returns an error
		if postStopCode, err := run(getConfig().postStopCmd); err != nil {
			os.Exit(postStopCode)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
}

// Set up polling functions and write their quit channels
// back to our Config
func handlePolling(config *Config) {
	var quit []chan bool
	for _, backend := range config.Backends {
		quit = append(quit, poll(backend))
	}
	for _, service := range config.Services {
		quit = append(quit, poll(service))
	}
	if config.Telemetry != nil {
		for _, sensor := range config.Telemetry.Sensors {
			quit = append(quit, poll(sensor))
		}
		go config.Telemetry.Serve()
	}
	config.QuitChannels = quit
}

// Every `pollTime` seconds, run the `PollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(pollable Pollable) chan bool {
	ticker := time.NewTicker(time.Duration(pollable.PollTime()) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !inMaintenanceMode() {
					pollable.PollAction()
				}
			case <-quit:
				return
			}
		}
	}()
	return quit
}

// Executes the given command and blocks until completed
func executeAndWait(cmd *exec.Cmd) (int, error) {
	if cmd == nil {
		return 0, nil
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), err
			}
		}
		// Should only happen if we misconfigure or there's some more
		// serious problem with the underlying open/exec syscalls. But
		// we'll let the lack of heartbeat tell us if something has gone
		// wrong to that extent.
		log.Errorln(err)
		return 1, err
	}
	return 0, nil
}

// Executes the given command and blocks until completed
// Returns the exit code and error message (if any).
// Logs errors
func run(cmd *exec.Cmd) (int, error) {
	code, err := executeAndWait(cmd)
	if err != nil {
		log.Errorln(err)
	}
	return code, err
}
