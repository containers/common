package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	cachedConfigError error
	cachedConfigMutex sync.Mutex
	cachedConfig      *Config
)

// Options to use when loading a Config via New().
type Options struct {
	// Set the loaded config as the default one which can later on be
	// accessed via Default().
	SetDefault bool

	// Additional configs that will be loaded after all system/user configs
	// and environment variables but the _OVERRIDE one.
	additionalConfigs []string
}

// New returns a Config as described in the containers.conf(5) man page.
func New(options *Options) (*Config, error) {
	if options == nil {
		options = &Options{}
	}
	cachedConfigMutex.Lock()
	defer cachedConfigMutex.Unlock()
	return newLocked(options)
}

// A helper function for New() expecting the caller to hold the
// cachedConfigMutex.
func newLocked(options *Options) (*Config, error) {
	// The _OVERRIDE variable _must_ always win.  That's a contract we need
	// to honor (for the Podman CI).
	if path := os.Getenv("CONTAINERS_CONF_OVERRIDE"); path != "" {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("CONTAINERS_CONF_OVERRIDE file: %w", err)
		}
		options.additionalConfigs = append(options.additionalConfigs, path)
	}

	// Start with the built-in defaults
	config, err := defaultConfig()
	if err != nil {
		return nil, err
	}

	// Now, gather the system configs and merge them as needed.
	configs, err := systemConfigs()
	if err != nil {
		return nil, fmt.Errorf("finding config on system: %w", err)
	}
	for _, path := range configs {
		// Merge changes in later configs with the previous configs.
		// Each config file that specified fields, will override the
		// previous fields.
		if err = readConfigFromFile(path, config); err != nil {
			return nil, fmt.Errorf("reading system config %q: %w", path, err)
		}
		logrus.Debugf("Merged system config %q", path)
		logrus.Tracef("%+v", config)
	}

	// If the caller specified a config path to use, then we read it to
	// override the system defaults.
	for _, add := range options.additionalConfigs {
		if add == "" {
			continue
		}
		// readConfigFromFile reads in container config in the specified
		// file and then merge changes with the current default.
		if err := readConfigFromFile(add, config); err != nil {
			return nil, fmt.Errorf("reading additional config %q: %w", add, err)
		}
		logrus.Debugf("Merged additional config %q", add)
		logrus.Tracef("%+v", config)
	}
	config.addCAPPrefix()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if err := config.setupEnv(); err != nil {
		return nil, err
	}

	if options.SetDefault {
		cachedConfig = config
		cachedConfigError = nil
	}

	return config, nil
}

// NewConfig creates a new Config. It starts with an empty config and, if
// specified, merges the config at `userConfigPath` path.
//
// Deprecated: use new instead.
func NewConfig(userConfigPath string) (*Config, error) {
	return New(&Options{additionalConfigs: []string{userConfigPath}})
}

// Default returns the default container config.  If no default config has been
// set yet, a new config will be loaded by New() and set as the default one.
// All callers are expected to use the returned Config read only.  Changing
// data may impact other call sites.
func Default() (*Config, error) {
	cachedConfigMutex.Lock()
	defer cachedConfigMutex.Unlock()
	if cachedConfig != nil || cachedConfigError != nil {
		return cachedConfig, cachedConfigError
	}
	cachedConfig, cachedConfigError = newLocked(&Options{SetDefault: true})
	return cachedConfig, cachedConfigError
}
