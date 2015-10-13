package main

import (
	// consul "github.com/hashicorp/consul/api"
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// these are populated by the flag variables
var (
	pollTime        int    // interval to poll in seconds
	healthCheckExec string // executable to run to check the health of the application
	changeExec      string // executable to run if the source of truth changes
)

func main() {
	parseArgs()
	healthQuit := poll(checkHealth, healthCheckExec)
	changeQuit := poll(checkForChanges, changeExec)

	// gracefully clean up so that our docker logs aren't cluttered after an exit 0
	// TODO: do we really need this?
	defer func() {
		close(healthQuit)
		close(changeQuit)
	}()

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		code, err := run(flag.Args()...)
		if err != nil {
			log.Println(err)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
}

func parseArgs() {
	flag.IntVar(&pollTime, "poll", 10, "Number of seconds to wait between polling health check")
	flag.StringVar(&healthCheckExec, "health", "", "Executable to run to check the health of the application.")
	flag.Parse()
}

type pollingFunc func(...string)

// Every `pollTime` seconds, run the `pollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(fn pollingFunc, args ...string) chan bool {
	ticker := time.NewTicker(time.Duration(pollTime) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				fn(args...)
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
func checkHealth(args ...string) {
	if code, _ := run(args...); code == 0 {
		writeHealthCheck()
	}
}

// Implements `pollingFunc`; args are the executable we run if the values in
// the central store have changed since the last run.
func checkForChanges(args ...string) {
	if upstreamChanged() {
		run(args...)
	}
}

// TODO: dummy implementations
func writeHealthCheck() {
	log.Printf("health check is ok!")
}

// TODO: dummy implementations
func upstreamChanged() bool {
	log.Printf("no change!")
	return false
}

// Runs an arbitrary string of arguments as an executable and its arguments.
// Returns the exit code and error message (if any).
func run(args ...string) (int, error) {
	cmd := exec.Command(args[:1][0], args[1:]...)
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
