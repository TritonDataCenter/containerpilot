package services

// NewPostStopConfig ...
func NewPostStopConfig(serviceName string, raw interface{}) (*Config, error) {
	if raw == nil {
		return nil, nil
	}
	service := &Config{Name: serviceName + ".postStop", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
