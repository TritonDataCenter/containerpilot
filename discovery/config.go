package discovery

import (
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/joyent/containerpilot/utils"
)

func configFromMap(raw map[string]interface{}) (*api.Config, error) {
	config := &struct {
		Address string `mapstructure:"address"`
		Scheme  string `mapstructure:"scheme"`
		Token   string `mapstructure:"token"`
	}{}
	if err := utils.DecodeRaw(raw, config); err != nil {
		return nil, err
	}
	return &api.Config{
		Address: config.Address,
		Scheme:  config.Scheme,
		Token:   config.Token,
	}, nil
}

func configFromURI(uri string) (*api.Config, error) {
	address, scheme := parseRawURI(uri)
	return &api.Config{
		Address: address,
		Scheme:  scheme,
	}, nil
}

// Returns the uri broken into an address and scheme portion
func parseRawURI(raw string) (string, string) {

	var scheme = "http" // default
	var address = raw   // we accept bare address w/o a scheme

	// strip the scheme from the prefix and (maybe) set the scheme to https
	if strings.HasPrefix(raw, "http://") {
		address = strings.Replace(raw, "http://", "", 1)
	} else if strings.HasPrefix(raw, "https://") {
		address = strings.Replace(raw, "https://", "", 1)
		scheme = "https"
	}
	return address, scheme
}
