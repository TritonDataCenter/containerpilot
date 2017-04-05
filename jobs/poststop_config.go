package jobs

// NewPostStopConfig ... (temporary, remove by end of v3)
func NewPostStopConfig(jobName string, raw interface{}) (*Config, error) {
	if raw == nil {
		return nil, nil
	}
	job := &Config{Name: jobName + ".postStop", Exec: raw}
	if err := job.Validate(nil); err != nil {
		return nil, err
	}
	return job, nil
}
