package jobs

// NewPreStartConfig ... (temporary, remove by end of v3)
func NewPreStartConfig(jobName string, raw interface{}) (*Config, error) {
	if raw == nil {
		return nil, nil
	}
	job := &Config{Name: jobName + ".preStart", Exec: raw}
	if err := job.Validate(nil); err != nil {
		return nil, err
	}
	return job, nil
}
