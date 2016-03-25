package containerbuddy

// Pollable is base abstraction for backends and services that support polling
type Pollable interface {
	PollTime() int
	PollAction()
}
