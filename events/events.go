// Package events contains the internal message bus used to broadcast
// events between goroutines representing jobs, watches, etc.
package events

import (
	"fmt"
)

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
	Metric
	Startup  // fired once after events are set up and event loop is started
	Shutdown // fired once after all jobs exit or on receiving SIGTERM
)

// global events
var (
	GlobalStartup          = Event{Code: Startup, Source: "global"}
	GlobalShutdown         = Event{Code: Shutdown, Source: "global"}
	NonEvent               = Event{Code: None, Source: ""}
	GlobalEnterMaintenance = Event{Code: EnterMaintenance, Source: "global"}
	GlobalExitMaintenance  = Event{Code: ExitMaintenance, Source: "global"}
)

// FromString parses a string as an EventCode enum
func FromString(codeName string) (EventCode, error) {
	switch codeName {
	case "exitSuccess":
		return ExitSuccess, nil
	case "exitFailed":
		return ExitFailed, nil
	case "stopping":
		return Stopping, nil
	case "stopped":
		return Stopped, nil
	case "healthy":
		return StatusHealthy, nil
	case "unhealthy":
		return StatusUnhealthy, nil
	case "changed":
		return StatusChanged, nil
	case "timerExpired":
		return TimerExpired, nil // end-users shouldn't use this in configs
	case "enterMaintenance":
		return EnterMaintenance, nil
	case "exitMaintenance":
		return ExitMaintenance, nil
	case "error":
		return Error, nil // end-users shouldn't use this in configs
	case "quit":
		return Quit, nil // end-users shouldn't use this in configs
	case "startup":
		return Startup, nil
	case "shutdown":
		return Shutdown, nil
	}
	return None, fmt.Errorf("%s is not a valid event code", codeName)
}
