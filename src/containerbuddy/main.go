package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func main() {

	config, configErr := loadConfig()
	if configErr != nil {
		log.Fatal(configErr)
	}

	// Run the onStart handler, if any, and exit if it returns an error
	if onStartCode, err := run(config.OnStart); err != nil {
		os.Exit(onStartCode)
	}

	// Set up signal handler for placing instance into maintenance mode
	handleSignals(config)

	var quit []chan bool
	for _, backend := range config.Backends {
		quit = append(quit, poll(backend, checkForChanges, backend.OnChangeExec))
	}
	for _, service := range config.Services {
		quit = append(quit, poll(service, checkHealth, service.HealthCheckExec))
	}
	config.QuitChannels = quit

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
		config.Command = getCmd(flag.Args())
		code, err := executeAndWait(config.Command)
		if err != nil {
			log.Println(err)
		}
		// Run the PostStop handler, if any, and exit if it returns an error
		if postStopCode, err := run(config.PostStop); err != nil {
			os.Exit(postStopCode)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
}

type pollingFunc func(Pollable, string)

// Every `pollTime` seconds, run the `pollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(config Pollable, fn pollingFunc, command string) chan bool {
	ticker := time.NewTicker(time.Duration(config.PollTime()) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !inMaintenanceMode() {
					fn(config, command)
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
func checkHealth(pollable Pollable, command string) {
	service := pollable.(*ServiceConfig) // if we pass a bad type here we crash intentionally
	if code, _ := run(command); code == 0 {
		service.SendHeartbeat()
	}
}

// Implements `pollingFunc`; args are the executable we run if the values in
// the central store have changed since the last run.
func checkForChanges(pollable Pollable, command string) {
	backend := pollable.(*BackendConfig) // if we pass a bad type here we crash intentionally
	if backend.CheckForUpstreamChanges() {
		run(command)
	}
}

func getCmd(args []string) *exec.Cmd {
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	} else {
		return exec.Command(args[0])
	}
}

// Executes the given command and blocks until completed
func executeAndWait(cmd *exec.Cmd) (int, error) {
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

// Runs an arbitrary string of arguments as an executable and its arguments.
// Returns the exit code and error message (if any).
func run(command string) (int, error) {
	if command != "" {
		args := strings.Split(command, " ")
		code, err := executeAndWait(getCmd(args))
		if err != nil {
			log.Println(err)
		}
		return code, err
	}
	return 0, nil
}
