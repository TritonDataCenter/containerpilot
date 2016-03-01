package containerbuddy

import (
	"flag"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

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

	// Run the onStart handler, if any, and exit if it returns an error
	if onStartCode, err := run(config.onStartCmd); err != nil {
		os.Exit(onStartCode)
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
		config.Command = argsToCmd(flag.Args())
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
		quit = append(quit, poll(backend, checkForChanges))
	}
	for _, service := range config.Services {
		quit = append(quit, poll(service, checkHealth))
	}
	config.QuitChannels = quit
}

type pollingFunc func(Pollable)

// Every `pollTime` seconds, run the `pollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(config Pollable, fn pollingFunc) chan bool {
	ticker := time.NewTicker(time.Duration(config.PollTime()) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !inMaintenanceMode() {
					fn(config)
				}
			case <-quit:
				return
			}
		}
	}()
	return quit
}

// Implements `pollingFunc`; args are the executable we use to check the
// application health and its arguments. If the error code on that exectable is
// 0, we write a TTL health check to the health check store.
func checkHealth(pollable Pollable) {
	service := pollable.(*ServiceConfig) // if we pass a bad type here we crash intentionally
	if code, _ := service.CheckHealth(); code == 0 {
		service.SendHeartbeat()
	}
}

// Implements `pollingFunc`; args are the executable we run if the values in
// the central store have changed since the last run.
func checkForChanges(pollable Pollable) {
	backend := pollable.(*BackendConfig) // if we pass a bad type here we crash intentionally
	if backend.CheckForUpstreamChanges() {
		backend.OnChange()
	}
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
