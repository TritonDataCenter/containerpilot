package main // import "github.com/joyent/containerpilot"

import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/core"
)

// Main executes the containerpilot CLI
func main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	subcommand, params := core.GetArgs()
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
	app.Run() // Blocks forever
}
