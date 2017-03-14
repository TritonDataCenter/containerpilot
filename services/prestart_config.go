package services

// NewPreStartConfig ...
func NewPreStartConfig(raw interface{}) (*ServiceConfig, error) {
	service := &ServiceConfig{Name: "preStart", Exec: raw}
	if err := service.Validate(nil); err != nil {
		return nil, err
	}
	return service, nil
}
