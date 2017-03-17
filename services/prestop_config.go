package services

// NewPreStopConfig ...
func NewPreStopConfig(serviceName string, raw interface{}) (*Config, error) {
	if raw == nil {
		return nil, nil
	}
	service := &Config{Name: serviceName + ".preStop", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
