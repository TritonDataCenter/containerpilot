package services

// NewPreStartConfig ...
func NewPreStartConfig(serviceName string, raw interface{}) (*Config, error) {
	if raw == nil || raw == "" {
		return nil, nil
	}
	service := &Config{Name: serviceName + ".preStart", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
