package core

import (
	"config"
	"flag"
	"os"
	"runtime"
	"time"
	"utils"

	log "github.com/Sirupsen/logrus"
)

// Main executes the containerbuddy CLI
func Main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	cfg, configErr := config.LoadConfig()
	if configErr != nil {
		log.Fatal(configErr)
	}

	// Run the preStart handler, if any, and exit if it returns an error
	if preStartCode, err := utils.Run(cfg.PreStartCmd); err != nil {
		os.Exit(preStartCode)
	}

	// Set up handlers for polling and to accept signal interrupts
	if 1 == os.Getpid() {
		reapChildren()
	}
	handleSignals(cfg)
	handlePolling(cfg)

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		cfg.Command = utils.ArgsToCmd(flag.Args())
		code, err := utils.ExecuteAndWait(cfg.Command)
		if err != nil {
			log.Errorln(err)
		}
		// Run the PostStop handler, if any, and exit if it returns an error
		if postStopCode, err := utils.Run(config.GetConfig().PostStopCmd); err != nil {
			os.Exit(postStopCode)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
}

// Set up polling functions and write their quit channels
// back to our config
func handlePolling(cfg *config.Config) {
	var quit []chan bool
	for _, backend := range cfg.Backends {
		quit = append(quit, poll(backend))
	}
	for _, service := range cfg.Services {
		quit = append(quit, poll(service))
	}
	if cfg.Telemetry != nil {
		for _, sensor := range cfg.Telemetry.Sensors {
			quit = append(quit, poll(sensor))
		}
		go cfg.Telemetry.Serve()
	}
	cfg.QuitChannels = quit
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
