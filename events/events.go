package events

// Event represents a single message in the EventBus
type Event struct {
	Code   EventCode
	Source string
}

// go:generate stringer -type EventCode

// EventCode is an enum for Events
type EventCode int

// EventCode enum
const (
	None        EventCode = iota // placeholder nil-event
	ExitSuccess                  // emitted when a Runner's exec completes with 0 exit code
	ExitFailed                   // emitted when a Runner's exec completes with non-0 exit code
	Stopping                     // emitted when a Runner is about to stop
	Stopped                      // emitted when a Runner has stopped
	StatusHealthy
	StatusUnhealthy
	StatusChanged
	TimerExpired
	EnterMaintenance
	ExitMaintenance
	Error
	Quit
	Startup  // fired once after events are set up and event loop is started
	Shutdown // fired once after all jobs exit or on receiving SIGTERM
)

// global events
var (
	GlobalStartup  = Event{Code: Startup, Source: "global"}
	GlobalShutdown = Event{Code: Shutdown, Source: "global"}
	QuitByClose    = Event{Code: Quit, Source: "closed"}
	NonEvent       = Event{Code: None, Source: ""}
)
