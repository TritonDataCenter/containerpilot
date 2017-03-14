package services

// NewPostStopConfig ...
func NewPostStopConfig(raw interface{}) (*ServiceConfig, error) {
	service := &ServiceConfig{Name: "postStop", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
