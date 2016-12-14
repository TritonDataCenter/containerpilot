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

	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/config"
	"github.com/joyent/containerpilot/coprocesses"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/tasks"
	"github.com/joyent/containerpilot/telemetry"

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
	ServiceBackend discovery.ServiceBackend
	Services       []*services.Service
	Backends       []*backends.Backend
	Tasks          []*tasks.Task
	Coprocesses    []*coprocesses.Coprocess
	Telemetry      *telemetry.Telemetry
	PreStartCmd    *commands.Command
	PreStopCmd     *commands.Command
	PostStopCmd    *commands.Command
	Command        *commands.Command
	StopTimeout    int
	QuitChannels   []chan bool
	maintModeLock  *sync.RWMutex
	signalLock     *sync.RWMutex
	paused         bool
	ConfigFlag     string
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

	if !flag.Parsed() {
		flag.StringVar(&configFlag, "config", "",
			"JSON config or file:// path to JSON config file.")
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
	cfg, err := config.ParseConfig(configFlag)
	if err != nil {
		return nil, err
	}
	if err = cfg.InitLogging(); err != nil {
		return nil, err
	}
	if log.GetLevel() >= log.DebugLevel {
		configJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Errorf("Error marshalling config for debug: %v", err)
		}
		log.Debugf("Loaded config: %v", string(configJSON))
	}
	a.PreStartCmd = cfg.PreStart
	a.PreStopCmd = cfg.PreStop
	a.PostStopCmd = cfg.PostStop
	a.StopTimeout = cfg.StopTimeout
	a.ServiceBackend = cfg.ServiceBackend
	a.Services = cfg.Services
	a.Backends = cfg.Backends
	a.Tasks = cfg.Tasks
	a.Coprocesses = cfg.Coprocesses
	a.Telemetry = cfg.Telemetry
	a.ConfigFlag = configFlag

	// set an environment variable for each service IP address so that
	// forked processes have access to this information
	for _, service := range a.Services {
		envKey := getEnvVarNameFromService(service.Name)
		os.Setenv(envKey, service.IPAddress)
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
	args := getArgs(flag.Args())
	cmd, err := commands.NewCommand(args, "0")
	if err != nil {
		log.Errorf("Unable to parse command arguments: %v", err)
	}
	cmd.Name = "APP"
	a.Command = cmd

	a.handleSignals()

	if a.PreStartCmd != nil {
		// Run the preStart handler, if any, and exit if it returns an error
		fields := log.Fields{"process": "PreStart"}
		if code, err := commands.RunAndWait(a.PreStartCmd, fields); err != nil {
			os.Exit(code)
		}
	}
	a.handleCoprocesses()
	a.handlePolling()

	if a.Command != nil {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		code, err := commands.RunAndWait(a.Command, nil)
		if err != nil {
			log.Println(err)
		}
		// Run the PostStop handler, if any, and exit if it returns an error
		if a.PostStopCmd != nil {
			fields := log.Fields{"process": "PostStop"}
			if postStopCode, err := commands.RunAndWait(a.PostStopCmd, fields); err != nil {
				os.Exit(postStopCode)
			}
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
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
		a.forAllServices(markServiceForMaintenance)
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
	a.stopPolling()
	a.forAllServices(deregisterService)

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

func (a *App) stopPolling() {
	for _, quit := range a.QuitChannels {
		quit <- true
	}
}

func markServiceForMaintenance(service *services.Service) {
	log.Infof("Marking for maintenance: %s", service.Name)
	service.MarkForMaintenance()
}

func deregisterService(service *services.Service) {
	log.Infof("Deregistering service: %s", service.Name)
	service.Deregister()
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

	a.stopPolling()
	a.forAllServices(deregisterService)
	a.stopCoprocesses()

	a.load(newApp)
	return nil
}

func (a *App) load(newApp *App) {
	a.ServiceBackend = newApp.ServiceBackend
	a.PostStopCmd = newApp.PostStopCmd
	a.PreStopCmd = newApp.PreStopCmd
	a.Services = newApp.Services
	a.Backends = newApp.Backends
	a.StopTimeout = newApp.StopTimeout
	if a.Telemetry != nil {
		a.Telemetry.Shutdown()
	}
	a.Telemetry = newApp.Telemetry
	a.Tasks = newApp.Tasks
	a.Coprocesses = newApp.Coprocesses
	a.handlePolling()
	a.handleCoprocesses()
}

type serviceFunc func(service *services.Service)

func (a *App) forAllServices(fn serviceFunc) {
	for _, service := range a.Services {
		fn(service)
	}
}

// HandlePolling sets up polling functions and write their quit channels
// back to our config
func (a *App) handlePolling() {
	var quit []chan bool
	for _, backend := range a.Backends {
		quit = append(quit, a.poll(backend))
	}
	for _, service := range a.Services {
		quit = append(quit, a.poll(service))
	}
	if a.Telemetry != nil {
		for _, sensor := range a.Telemetry.Sensors {
			quit = append(quit, a.poll(sensor))
		}
		a.Telemetry.Serve()
	}
	if a.Tasks != nil {
		for _, task := range a.Tasks {
			quit = append(quit, a.poll(task))
		}
	}
	a.QuitChannels = quit
}

func (a *App) handleCoprocesses() {
	for _, coprocess := range a.Coprocesses {
		go coprocess.Start()
	}
}

func (a *App) stopCoprocesses() {
	for _, coprocess := range a.Coprocesses {
		coprocess.Stop()
	}
}
