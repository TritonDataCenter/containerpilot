package main // import "github.com/joyent/containerpilot"

import (
	"os"
	"runtime"

	"github.com/joyent/containerpilot/core"
	"github.com/joyent/containerpilot/sup"

	log "github.com/sirupsen/logrus"

	// Import backends so that they initialize
	_ "github.com/joyent/containerpilot/discovery/consul"
	_ "github.com/joyent/containerpilot/discovery/etcd"
)

// Main executes the containerpilot CLI
func main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	// If we're running as PID1, we fork and run as a supervisor
	// so that we can cleanly handle reaping of child processes.
	// We fork before doing *anything* else so we don't have to
	// worry about where any new threads spawned by the runtime.
	if os.Getpid() == 1 {
		sup.Run() // blocks forever
		return
	}

	app, configErr := core.LoadApp()
	if configErr != nil {
		log.Fatal(configErr)
	}
	app.Run() // Blocks forever
}
