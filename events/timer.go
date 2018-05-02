package events

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

// NewEventTimeout starts a goroutine on a timer that will send a
// TimerExpired event when the timer expires
func NewEventTimeout(
	ctx context.Context,
	rx chan Event,
	tick time.Duration,
	name string,
) {
	go func() {
		timeout := time.After(tick)
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			// sending the timeout event potentially races with a closing
			// rx channel, so just recover from the panic and exit
			defer func() {
				if r := recover(); r != nil {
					return
				}
			}()
			event := Event{Code: TimerExpired, Source: name}
			log.Debugf("timeout: %v", event)
			rx <- event
		}
	}()
}

// NewEventTimer starts a goroutine with a timer that will send a
// TimerExpired event every time the timer expires
func NewEventTimer(
	ctx context.Context,
	rx chan Event,
	tick time.Duration,
	name string,
) {
	go func() {
		ticker := time.NewTicker(tick)
		// sending the timeout event potentially races with a closing
		// rx channel, so just recover from the panic and exit
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				event := Event{Code: TimerExpired, Source: name}
				// do not log the telemetry health check timer ticks since this
				// log statement is called once for every internal heartbeat
				// check, which is a bit excessive under DEBUG logging [GH-556]
				if event.Source != "containerpilot.heartbeat" {
					log.Debugf("timer: %v", event)
				}
				rx <- event
			}
		}
	}()
}
