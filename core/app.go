package core

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joyent/containerpilot/config"
	"github.com/joyent/containerpilot/control"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/jobs"
	"github.com/joyent/containerpilot/subcommands"
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
type App struct {
	ControlServer *control.HTTPServer
	Discovery     discovery.Backend
	Jobs          []*jobs.Job
	Watches       []*watches.Watch
	Telemetry     *telemetry.Telemetry
	StopTimeout   int
	signalLock    *sync.RWMutex
	ConfigFlag    string
	Bus           *events.EventBus
}

// EmptyApp creates an empty application
func EmptyApp() *App {
	app := &App{}
	app.signalLock = &sync.RWMutex{}
	return app
}

// LoadApp parses the commandline arguments and loads the config
func LoadApp() (*App, error) {

	var versionFlag bool
	var templateFlag bool
	var reloadFlag bool

	var configFlag string
	var renderFlag string
	var maintFlag string

	var putMetricFlags MultiFlag
	var putEnvFlags MultiFlag

	if !flag.Parsed() {
		flag.BoolVar(&versionFlag, "version", false,
			"Show version identifier and quit.")

		flag.BoolVar(&templateFlag, "template", false,
			"Render template and quit. (default: false)")

		flag.BoolVar(&reloadFlag, "reload", false,
			"reload a ContainerPilot process through its control socket.")

		flag.StringVar(&configFlag, "config", "",
			"file path to JSON5 configuration file.")

		flag.StringVar(&renderFlag, "out", "-",
			"-(default) for stdout or file path where to save rendered JSON config file.")

		flag.StringVar(&maintFlag, "maintenance", "",
			"enable/disable maintanence mode through a ContainerPilot process control socket.")

		flag.Var(&putMetricFlags, "putmetric",
			"update metrics of a ContainerPilot process through its control socket.")

		flag.Var(&putEnvFlags, "putenv",
			"update environ of a ContainerPilot process through its control socket.")

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

	cmd, err := subcommands.Init(configFlag)
	if err != nil {
		fmt.Println("Subcommands: failed to init config:", err)
		os.Exit(2)
	}

	if reloadFlag {
		if err := cmd.SendReload(); err != nil {
			fmt.Println("Reload: failed to run subcommand:", err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	if maintFlag != "" {
		if err := cmd.SendMaintenance(maintFlag); err != nil {
			fmt.Println("SendMaintenance: failed to run subcommand:", err)
			os.Exit(2)
		}

		os.Exit(0)
	}

	if putEnvFlags.Len() != 0 {
		if err := cmd.SendEnviron(putEnvFlags.Values); err != nil {
			fmt.Println("SendEnviron: failed to run subcommand:", err)
			os.Exit(2)
		}

		os.Exit(0)
	}

	if putMetricFlags.Len() != 0 {
		if err := cmd.SendMetric(putMetricFlags.Values); err != nil {
			fmt.Println("SendMetric: failed to run subcommand:", err)
			os.Exit(2)
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

	if err := cfg.InitLogging(); err != nil {
		return nil, err
	}
	if log.GetLevel() >= log.DebugLevel {
		configJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Errorf("error marshalling config for debug: %v", err)
		}
		log.Debugf("loaded config: %v", string(configJSON))
	}

	cs, err := control.NewHTTPServer(cfg.Control)
	if err != nil {
		return nil, err
	}
	a.ControlServer = cs

	a.StopTimeout = cfg.StopTimeout
	a.Discovery = cfg.Discovery
	a.Jobs = jobs.FromConfigs(cfg.Jobs)
	a.Watches = watches.FromConfigs(cfg.Watches)
	a.Telemetry = telemetry.NewTelemetry(cfg.Telemetry)
	a.ConfigFlag = configFlag // stash the old config

	// set an environment variable for each job IP address so that
	// forked processes have access to this information
	for _, job := range a.Jobs {
		if job.Service != nil {
			envKey := getEnvVarNameFromService(job.Name)
			os.Setenv(envKey, job.Service.IPAddress)
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
	for {
		a.Bus = events.NewEventBus()
		a.ControlServer.Run(a.Bus)
		a.handleSignals()
		a.handlePolling()
		if !a.Bus.Wait() {
			break
		}
		if err := a.reload(); err != nil {
			break
		}
	}
}

// Render the command line args thru golang templating so we can
// interpolate environment variables
func getArgs(args []string) []string {
	var renderedArgs []string
	for _, arg := range args {
		newArg, err := config.ApplyTemplate([]byte(arg))
		if err != nil {
			log.Errorf("unable to render command arguments template: %v", err)
			renderedArgs = args // skip rendering on error
			break
		}
		renderedArgs = append(renderedArgs, string(newArg))
	}
	return renderedArgs
}

// Terminate kills the application
func (a *App) Terminate() {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	a.Bus.Shutdown()
	if a.StopTimeout > 0 {
		time.AfterFunc(time.Duration(a.StopTimeout)*time.Second, func() {
			for _, job := range a.Jobs {
				log.Infof("killing processes for job %#v", job.Name)
				job.Kill()
			}
		})
		return
	}
	for _, job := range a.Jobs {
		log.Infof("killing processes for job %#v", job.Name)
		job.Kill()
	}
}

// reload does the actual work of reloading the configuration and
// updating the App with those changes. The EventBus should be
// already shut down before we call this.
func (a *App) reload() error {
	newApp, err := NewApp(a.ConfigFlag)
	if err != nil {
		log.Errorf("error initializing config: %v", err)
		return err
	}
	a.Discovery = newApp.Discovery
	a.Jobs = newApp.Jobs
	a.Watches = newApp.Watches
	a.StopTimeout = newApp.StopTimeout
	a.Telemetry = newApp.Telemetry
	a.ControlServer = newApp.ControlServer
	return nil
}

// HandlePolling sets up polling functions and write their quit channels
// back to our config
func (a *App) handlePolling() {
	for _, job := range a.Jobs {
		job.Run(a.Bus)
	}
	for _, watch := range a.Watches {
		watch.Run(a.Bus)
	}
	if a.Telemetry != nil {
		for _, sensor := range a.Telemetry.Sensors {
			sensor.Run(a.Bus)
		}
		a.Telemetry.Serve()
	}
	// kick everything off
	a.Bus.Publish(events.GlobalStartup)
}
