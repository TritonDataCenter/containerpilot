package main // import "github.com/joyent/containerpilot"

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/joyent/containerpilot/config"
	"github.com/joyent/containerpilot/core"
	"github.com/joyent/containerpilot/utils"
)

// Main executes the containerpilot CLI
func main() {
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
		core.ReapChildren()
	}
	core.HandleSignals(cfg)
	core.HandlePolling(cfg)

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		cfg.Command = utils.ArgsToCmd(flag.Args())
		code, err := utils.ExecuteAndWait(cfg.Command)
		if err != nil {
			log.Println(err)
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
