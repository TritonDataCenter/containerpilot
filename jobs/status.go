package jobs

// JobStatus is an enum of job health status
type JobStatus int

// JobStatus enum
const (
	statusIdle JobStatus = iota // will be default value before starting
	statusUnknown
	statusHealthy
	statusUnhealthy
	statusMaintenance
	statusAlwaysHealthy
)

func (i JobStatus) String() string {
	switch i {
	case 2:
		return "healthy"
	case 3:
		return "unhealthy"
	case 4:
		return "maintenance"
	case 5:
		// for hardcoded "always healthy" jobs
		return "healthy"
	default:
		// both idle and unknown return unknown for purposes of serialization
		return "unknown"
	}
}
