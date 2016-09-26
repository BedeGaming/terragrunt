package config

import (
	"fmt"
	"io/ioutil"

	"github.com/gruntwork-io/terragrunt/errors"
	"github.com/gruntwork-io/terragrunt/locks"
	"github.com/gruntwork-io/terragrunt/remote"
	"github.com/hashicorp/hcl"
)

const configFilePath = ".terragrunt"

// Config represents a parsed and expanded configuration
type Config struct {
	Lock        locks.Lock
	RemoteState *remote.RemoteState
}

// fileConfig represents the configuration supported in the .terragrunt file
type fileConfig struct {
	Lock        *LockConfig `json:"lock,omitempty"`
	RemoteState *remote.RemoteState
}

// LockConfig represents generic configuration for Lock providers
type LockConfig struct {
	Backend string            `json:"backend"`
	Config  map[string]string `json:"config"`
}

// Read the Terragrunt config file from its default location
func Read() (*Config, error) {
	return parseConfigFile(configFilePath)
}

// Parse the Terragrunt config file at the given path
func parseConfigFile(configPath string) (*Config, error) {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.WithStackTraceAndPrefix(err, "Error reading Terragrunt config file %s", configPath)
	}

	config, err := parseConfigString(string(bytes))
	if err != nil {
		return nil, errors.WithStackTraceAndPrefix(err, "Error parsing Terragrunt config file %s", configPath)
	}

	return config, nil
}

// Parse the Terragrunt config contained in the given string
func parseConfigString(configSrc string) (*Config, error) {
	f := &fileConfig{}
	if err := hcl.Decode(f, configSrc); err != nil {
		return nil, errors.WithStackTrace(err)
	}

	c := &Config{}

	if lconf := f.Lock; lconf != nil {
		lock, err := lookupLock(lconf.Backend, lconf.Config)
		if err != nil {
			return nil, fmt.Errorf("unable to configure lock %s: %s", lconf.Backend, err)
		}

		c.Lock = lock
	}

	if f.RemoteState != nil {
		f.RemoteState.FillDefaults()
		if err := f.RemoteState.Validate(); err != nil {
			return nil, err
		}

		c.RemoteState = f.RemoteState
	}

	return c, nil
}
