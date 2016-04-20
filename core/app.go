package core

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/config"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/tasks"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/utils"

	log "github.com/Sirupsen/logrus"
)

var (
	// Version is the version for this build, set at build time via LDFLAGS
	Version string
	// GitHash is the short-form commit hash of this build, set at build time
	GitHash string
)

// App ...
type App struct {
	DiscoveryService discovery.DiscoveryService
	Services         []*services.Service
	Backends         []*backends.Backend
	Tasks            []*tasks.Task
	Telemetry        *telemetry.Telemetry
	PreStartCmd      *exec.Cmd
	PreStopCmd       *exec.Cmd
	PostStopCmd      *exec.Cmd
	Command          *exec.Cmd
	StopTimeout      int
	QuitChannels     []chan bool
	maintModeLock    *sync.RWMutex
	signalLock       *sync.RWMutex
	paused           bool
	ConfigFlag       string
}

// NewApp ...
func NewApp() *App {
	app := &App{}
	app.maintModeLock = &sync.RWMutex{}
	app.signalLock = &sync.RWMutex{}
	return app
}

// LoadApp ...
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

	cfg, err := config.ParseConfig(configFlag)
	if err != nil {
		return nil, err
	}
	app, err := InitializeApp(cfg)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// InitializeApp creates a new App from the config
func InitializeApp(cfg *config.Config) (*App, error) {
	a := NewApp()

	// onStart has been deprecated for preStart. Remove in 2.0
	if cfg.PreStart != nil && cfg.OnStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use only the preStart option provided")
	}

	// alias the onStart behavior to preStart
	if cfg.PreStart == nil && cfg.OnStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use the onStart option as a preStart")
		cfg.PreStart = cfg.OnStart
	}

	preStartCmd, err := cfg.ParsePreStart()
	if err != nil {
		return nil, err
	}
	a.PreStartCmd = preStartCmd

	preStopCmd, err := cfg.ParsePreStop()
	if err != nil {
		return nil, err
	}
	a.PreStopCmd = preStopCmd

	postStopCmd, err := cfg.ParsePostStop()
	if err != nil {
		return nil, err
	}
	a.PostStopCmd = postStopCmd

	stopTimeout, err := cfg.ParseStopTimeout()
	if err != nil {
		return nil, err
	}
	a.StopTimeout = stopTimeout

	discoveryService, err := cfg.ParseDiscoveryService()
	if err != nil {
		return nil, err
	}
	a.DiscoveryService = discoveryService

	backends, err := cfg.ParseBackends(discoveryService)
	if err != nil {
		return nil, err
	}
	a.Backends = backends

	services, err := cfg.ParseServices(discoveryService)
	if err != nil {
		return nil, err
	}
	a.Services = services

	telemetry, err := cfg.ParseTelemetry()
	if err != nil {
		return nil, err
	}

	if telemetry != nil {
		telemetryService, err2 := config.CreateTelemetryService(telemetry, discoveryService)
		if err2 != nil {
			return nil, err2
		}
		a.Telemetry = telemetry
		a.Services = append(a.Services, telemetryService)
	}

	tasks, err := cfg.ParseTasks()
	if err != nil {
		return nil, err
	}
	a.Tasks = tasks

	return a, nil
}

// Run ...
func (a *App) Run() {
	// Run the preStart handler, if any, and exit if it returns an error
	if preStartCode, err := utils.RunWithFields(a.PreStartCmd, log.Fields{"process": "PreStart"}); err != nil {
		os.Exit(preStartCode)
	}

	// Set up handlers for polling and to accept signal interrupts
	if 1 == os.Getpid() {
		ReapChildren()
	}
	a.HandleSignals()
	a.HandlePolling()

	if len(flag.Args()) != 0 {
		// Run our main application and capture its stdout/stderr.
		// This will block until the main application exits and then os.Exit
		// with the exit code of that application.
		a.Command = utils.ArgsToCmd(flag.Args())
		a.Command.Stderr = os.Stderr
		a.Command.Stdout = os.Stdout
		code, err := utils.ExecuteAndWait(a.Command)
		if err != nil {
			log.Println(err)
		}
		// Run the PostStop handler, if any, and exit if it returns an error
		if postStopCode, err := utils.RunWithFields(a.PostStopCmd, log.Fields{"process": "PostStop"}); err != nil {
			os.Exit(postStopCode)
		}
		os.Exit(code)
	}

	// block forever, as we're polling in the two polling functions and
	// did not os.Exit by waiting on an external application.
	select {}
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

	// Run and wait for preStop command to exit
	utils.RunWithFields(a.PreStopCmd, log.Fields{"process": "PreStop"})

	cmd := a.Command
	if cmd == nil || cmd.Process == nil {
		// Not managing the process, so don't do anything
		return
	}
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
	log.Infof("Killing Process %#v", a.Command.Process)
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

// Reload ...
func (a *App) Reload() error {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	log.Infof("Reloading configuration.")

	newCfg, err := config.ParseConfig(a.ConfigFlag)
	if err != nil {
		return err
	}
	newApp, err := InitializeApp(newCfg)
	if err != nil {
		return err
	}

	a.stopPolling()
	a.forAllServices(deregisterService)
	signal.Reset()

	a.load(newApp)
	return nil
}

func (a *App) load(newApp *App) {
	a.DiscoveryService = newApp.DiscoveryService
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
	a.HandleSignals()
	a.HandlePolling()
}

type serviceFunc func(service *services.Service)

func (a *App) forAllServices(fn serviceFunc) {
	for _, service := range a.Services {
		fn(service)
	}
}

// HandlePolling sets up polling functions and write their quit channels
// back to our config
func (a *App) HandlePolling() {
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
