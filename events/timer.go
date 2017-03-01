package events

import (
	"context"
	"time"
)

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
			rx <- Event{Code: TimerExpired, Source: name}
		}
	}()
}

func NewEventTimer(
	ctx context.Context,
	rx chan Event,
	tick time.Duration,
	name string,
) {
	go func() {
		ticker := time.NewTicker(tick)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rx <- Event{Code: TimerExpired, Source: name}
		}
	}()
}
