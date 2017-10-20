// ContainerPilot is an init system for cloud-native distributed applications
// that automates the process of service discovery, configuration, and
// lifecycle management inside the container, so you can focus on your apps.
package main // import "github.com/joyent/containerpilot"

import (
	"os"
	"runtime"

	"github.com/joyent/containerpilot/core"
	"github.com/joyent/containerpilot/sup"
	log "github.com/sirupsen/logrus"
)

// Main executes the containerpilot CLI
func main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	subcommand, params := core.GetArgs()
	configPath := params.ConfigPath

	// If we're running as PID1, we fork and run as a supervisor
	// so that we can cleanly handle reaping of child processes.
	// We fork before doing *anything* else so we don't have to
	// worry about where any new threads spawned by the runtime.
	if os.Getpid() == 1 {
		sup.Run(configPath) // blocks forever
		return
	}

	if subcommand != nil {
		err := subcommand(params)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	app, configErr := core.NewApp(params.ConfigPath)
	if configErr != nil {
		log.Fatal(configErr)
	}
	app.Run() // blocks forever
}
