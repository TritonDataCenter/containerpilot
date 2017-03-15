package services

// NewPostStopConfig ...
func NewPostStopConfig(raw interface{}) (*ServiceConfig, error) {
	if raw == nil || raw == "" {
		return nil, nil
	}
	service := &ServiceConfig{Name: "postStop", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
