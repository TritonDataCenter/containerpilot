package services

// NewPreStopConfig ...
func NewPreStopConfig(raw interface{}) (*ServiceConfig, error) {
	service := &ServiceConfig{Name: "preStop", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
