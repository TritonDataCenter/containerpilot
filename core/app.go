package core

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joyent/containerpilot/checks"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/config"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/watches"

	log "github.com/Sirupsen/logrus"
)

var (
	// Version is the version for this build, set at build time via LDFLAGS
	Version string
	// GitHash is the short-form commit hash of this build, set at build time
	GitHash string
)

// App encapsulates the state of ContainerPilot after the initial setup.
// after it is run, it can be reloaded and paused with signals.
type App struct {
	Discovery     discovery.Backend
	Services      []*services.Service
	Checks        []*checks.HealthCheck
	Watches       []*watches.Watch
	Telemetry     *telemetry.Telemetry
	StopTimeout   int
	maintModeLock *sync.RWMutex // TODO: probably want to move this to Service.Status
	signalLock    *sync.RWMutex
	paused        bool
	ConfigFlag    string
	Bus           *events.EventBus
}

// EmptyApp creates an empty application
func EmptyApp() *App {
	app := &App{}
	app.maintModeLock = &sync.RWMutex{}
	app.signalLock = &sync.RWMutex{}
	return app
}

// LoadApp parses the commandline arguments and loads the config
func LoadApp() (*App, error) {

	var configFlag string
	var versionFlag bool
	var renderFlag string
	var templateFlag bool

	if !flag.Parsed() {
		flag.StringVar(&configFlag, "config", "",
			"JSON config or file:// path to JSON config file.")
		flag.BoolVar(&templateFlag, "template", false,
			"Render template and quit. (default: false)")
		flag.StringVar(&renderFlag, "out", "-",
			"-(default) for stdout or file:// path where to save rendered JSON config file.")
		flag.BoolVar(&versionFlag, "version", false, "Show version identifier and quit.")
		flag.Parse()
	}
	if versionFlag {
		fmt.Printf("Version: %s\nGitHash: %s\n", Version, GitHash)
		os.Exit(0)
	}
	if configFlag == "" {
		configFlag = os.Getenv("CONTAINERPILOT")
	}
	if templateFlag {

		err := config.RenderConfig(configFlag, renderFlag)
		if err != nil {
			return nil, err
		}
		os.Exit(0)
	}

	os.Setenv("CONTAINERPILOT_PID", fmt.Sprintf("%v", os.Getpid()))
	app, err := NewApp(configFlag)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// NewApp creates a new App from the config
func NewApp(configFlag string) (*App, error) {
	a := EmptyApp()
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return nil, err
	}

	// TODO: need to make a Service out of these too
	args := getArgs(flag.Args())
	cmd, err := commands.NewCommand(args, "0")
	if err != nil {
		log.Errorf("Unable to parse command arguments: %v", err)
	}

	if err = cfg.InitLogging(); err != nil {
		return nil, err
	}
	if log.GetLevel() >= log.DebugLevel {
		configJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Errorf("error marshalling config for debug: %v", err)
		}
		log.Debugf("loaded config: %v", string(configJSON))
	}
	a.StopTimeout = cfg.StopTimeout
	a.Discovery = cfg.Discovery
	a.Checks = cfg.Checks
	a.Services = cfg.Services
	a.Watches = cfg.Watches
	a.Telemetry = cfg.Telemetry
	a.ConfigFlag = configFlag

	// set an environment variable for each service IP address so that
	// forked processes have access to this information
	for _, service := range a.Services {
		if service.Definition != nil {
			envKey := getEnvVarNameFromService(service.Name)
			os.Setenv(envKey, service.Definition.IPAddress)
		}
	}

	return a, nil
}

// Normalize the validated service name as an environment variable
func getEnvVarNameFromService(service string) string {
	envKey := strings.ToUpper(service)
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = fmt.Sprintf("CONTAINERPILOT_%v_IP", envKey)
	return envKey
}

// Run starts the application and blocks until finished
func (a *App) Run() {
	// Set up handlers for polling and to accept signal interrupts
	if 1 == os.Getpid() {
		reapChildren()
	}
	a.Bus = events.NewEventBus()
	a.handleSignals()
	a.handlePolling()

	// block forever, as we're polling in the two polling functions
	select {}
}

// Render the command line args thru golang templating so we can
// interpolate environment variables
func getArgs(args []string) []string {
	var renderedArgs []string
	for _, arg := range args {
		newArg, err := config.ApplyTemplate([]byte(arg))
		if err != nil {
			log.Errorf("Unable to render command arguments template: %v", err)
			renderedArgs = args // skip rendering on error
			break
		}
		renderedArgs = append(renderedArgs, string(newArg))
	}
	return renderedArgs
}

// ToggleMaintenanceMode marks all services for maintenance
func (a *App) ToggleMaintenanceMode() {
	a.maintModeLock.RLock()
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	defer a.maintModeLock.RUnlock()
	a.paused = !a.paused
	if a.paused {
		a.Bus.Publish(events.Event{events.EnterMaintenance, events.Global})
	}
}

// InMaintenanceMode checks if the App is in maintenance mode
func (a *App) InMaintenanceMode() bool {
	// we wrap access to `paused` in a RLock so that if we're in the middle of
	// marking services for maintenance we don't get stale reads
	a.maintModeLock.RLock()
	defer a.maintModeLock.RUnlock()
	return a.paused
}

// Terminate kills the application
func (a *App) Terminate() {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	a.Bus.Shutdown()

	// Run and wait for preStop command to exit (continues
	// unconditionally so we don't worry about returned errors here)
	commands.RunAndWait(a.PreStopCmd, log.Fields{"process": "PreStop"})
	if a.Command == nil || a.Command.Cmd == nil ||
		a.Command.Cmd.Process == nil {
		// Not managing the process, so don't do anything
		return
	}
	cmd := a.Command.Cmd // get the underlying process
	if a.StopTimeout > 0 {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Warnf("Error sending SIGTERM to application: %s", err)
		} else {
			time.AfterFunc(time.Duration(a.StopTimeout)*time.Second, func() {
				log.Infof("Killing Process %#v", cmd.Process)
				cmd.Process.Kill()
			})
			return
		}
	}
	log.Infof("Killing Process %#v", a.Command.Cmd.Process)
	cmd.Process.Kill()
}

// Reload will try to update the running application by
// loading the config and applying changes to the services
// A reload cannot change the shimmed application, or the preStart script
func (a *App) Reload() error {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	log.Infof("Reloading configuration.")

	newApp, err := NewApp(a.ConfigFlag)
	if err != nil {
		log.Errorf("Error initializing config: %v", err)
		return err
	}

	a.Bus.Shutdown()
	a.load(newApp)
	return nil
}

func (a *App) load(newApp *App) {
	a.Discovery = newApp.Discovery
	a.Services = newApp.Services
	a.Checks = newApp.Checks
	a.StopTimeout = newApp.StopTimeout
	if a.Telemetry != nil {
		a.Telemetry.Shutdown()
	}
	a.Telemetry = newApp.Telemetry
	a.handlePolling()
}

// HandlePolling sets up polling functions and write their quit channels
// back to our config
func (a *App) handlePolling() {

	for _, service := range a.Services {
		service.Run(a.Bus)
	}
	for _, check := range a.Checks {
		check.Run(a.Bus)
	}
	for _, watch := range a.Watches {
		watch.Run(a.Bus)
	}
	if a.Telemetry != nil {
		for _, sensor := range a.Sensors {
			sensor.Run(a.Bus)
		}
		a.Telemetry.Serve()
	}
}
