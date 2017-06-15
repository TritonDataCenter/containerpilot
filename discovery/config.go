package discovery

import (
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/joyent/containerpilot/utils"
)

type parsedConfig struct {
	Address string `mapstructure:"address"`
	Scheme  string `mapstructure:"scheme"`
	Token   string `mapstructure:"token"`

	// optional TLS settings
	HTTPCAFile        string `mapstructure:"tlscafile"`
	HTTPCAPath        string `mapstructure:"tlscapath"`
	HTTPClientCert    string `mapstructure:"tlsclientcert"`
	HTTPClientKey     string `mapstructure:"tlsclientkey"`
	HTTPTLSServerName string `mapstructure:"tlsservername"`
	HTTPSSLVerify     bool   `mapstructure:"tlsverify"`
}

// override an already-parsed parsedConfig with any options that might
// be set in the environment and then return the TLSConfig
func getTLSConfig(parsed *parsedConfig) api.TLSConfig {
	if cafile := os.Getenv("CONSUL_CACERT"); cafile != "" {
		parsed.HTTPCAFile = cafile
	}
	if capath := os.Getenv("CONSUL_CAPATH"); capath != "" {
		parsed.HTTPCAPath = capath
	}
	if clientCert := os.Getenv("CONSUL_CLIENT_CERT"); clientCert != "" {
		parsed.HTTPClientCert = clientCert
	}
	if clientKey := os.Getenv("CONSUL_CLIENT_KEY"); clientKey != "" {
		parsed.HTTPClientKey = clientKey
	}
	if serverName := os.Getenv("CONSUL_TLS_SERVER_NAME"); serverName != "" {
		parsed.HTTPClientKey = serverName
	}
	verify := os.Getenv("CONSUL_HTTP_SSL_VERIFY")
	switch strings.ToLower(verify) {
	case "1", "true":
		parsed.HTTPSSLVerify = true
	case "0", "false":
		parsed.HTTPSSLVerify = false
	}
	tlsConfig := api.TLSConfig{
		Address:            parsed.HTTPTLSServerName,
		CAFile:             parsed.HTTPCAPath,
		CertFile:           parsed.HTTPCAPath,
		KeyFile:            parsed.HTTPClientKey,
		InsecureSkipVerify: !parsed.HTTPSSLVerify,
	}
	return tlsConfig
}

func configFromMap(raw map[string]interface{}) (*api.Config, error) {
	parsed := &parsedConfig{}
	if err := utils.DecodeRaw(raw, parsed); err != nil {
		return nil, err
	}
	config := &api.Config{
		Address:   parsed.Address,
		Scheme:    parsed.Scheme,
		Token:     parsed.Token,
		TLSConfig: getTLSConfig(parsed),
	}
	return config, nil
}

func configFromURI(uri string) (*api.Config, error) {
	address, scheme := parseRawURI(uri)
	parsed := &parsedConfig{Address: address, Scheme: scheme}
	config := &api.Config{
		Address:   parsed.Address,
		Scheme:    parsed.Scheme,
		Token:     parsed.Token,
		TLSConfig: getTLSConfig(parsed),
	}
	return config, nil
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
