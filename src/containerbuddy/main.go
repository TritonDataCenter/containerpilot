package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	config := loadConfig()
	var quit []chan bool
	for _, backend := range config.Backends {
		quit = append(quit, poll(backend, checkForChanges, backend.onChangeArgs))
	}
	for _, service := range config.Services {
		quit = append(quit, poll(service, checkHealth, service.healthArgs))
	}

	// gracefully clean up so that our docker logs aren't cluttered after an exit 0
	// TODO: do we really need this?
	defer func() {
		for _, ch := range quit {
			close(ch)
		}
	}()

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		code, err := run(flag.Args())
		if err != nil {
			log.Println(err)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
}

type pollingFunc func(Pollable, []string)

// Every `pollTime` seconds, run the `pollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(config Pollable, fn pollingFunc, args []string) chan bool {
	ticker := time.NewTicker(time.Duration(config.PollTime()) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				fn(config, args)
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
func checkHealth(pollable Pollable, args []string) {
	service := pollable.(*ServiceConfig) // if we pass a bad type here we crash intentionally
	if code, _ := run(args); code == 0 {
		service.WriteHealthCheck()
	}
}

// Implements `pollingFunc`; args are the executable we run if the values in
// the central store have changed since the last run.
func checkForChanges(pollable Pollable, args []string) {
	backend := pollable.(*BackendConfig) // if we pass a bad type here we crash intentionally
	if backend.CheckForUpstreamChanges() {
		run(args)
	}
}

// Runs an arbitrary string of arguments as an executable and its arguments.
// Returns the exit code and error message (if any).
func run(args []string) (int, error) {
	var cmd *exec.Cmd
	if len(args) > 1 {
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(args[0])
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), err
			}
		}
		// only happens if we misconfigure, so just die here
		log.Fatal(err)
	}
	return 0, nil
}
