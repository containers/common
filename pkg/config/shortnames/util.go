package shortnames

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/pkg/sysregistriesv2"

	"github.com/BurntSushi/toml"
	"github.com/containers/storage/pkg/unshare"
	"github.com/pkg/errors"
)

//// ALGORITHMIC UTILITY ////

// linear search returning location of element in a slice (returns -1 if not found)
func find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// removeIndex removes an element at an index in a slice
func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

// removeDuplicates removes duplicate values in given slice
func removeDuplicates(s []string) []string {
	for i := 0; i < len(s); i++ {
		for i2 := i + 1; i2 < len(s); i2++ {
			if s[i] == s[i2] {
				// delete
				s = append(s[:i2], s[i2+1:]...)
				i2--
			}
		}
	}
	return s
}

//// FILEPATH UTILITY ////

// The configuration files loaded here are used with 'user' at highest priority and 'share' least.

const (
	// _configPath is the path to the containers/shortnames.conf
	// inside a given config directory.
	_configPath = "containers/shortnames.conf"
	// DefaultShortnamesConfig holds the default shortnames config path
	DefaultShortnamesConfig = "/usr/share/" + _configPath
	// OverrideShortnamesConfig holds the default config paths overridden by the root user
	OverrideShortnamesConfig = "/etc/" + _configPath
	// UserOverrideShortnamesConfig holds the containers config path overridden by the rootless user
	UserOverrideShortnamesConfig = ".config/" + _configPath
)

// This allows the home directory to be used
func rootlessConfigPath() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, _configPath), nil
	}
	home, err := unshare.HomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, UserOverrideShortnamesConfig), nil
}

func systemConfigs() ([]string, error) {
	configs := []string{}
	path := os.Getenv("SHORTNAMES_CONF")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return nil, errors.Wrapf(err, "failed to stat of %s from SHORTNAMES_CONF environment variable", path)
		}
		return append(configs, path), nil
	}
	if _, err := os.Stat(DefaultShortnamesConfig); err == nil {
		configs = append(configs, DefaultShortnamesConfig)
	}
	if _, err := os.Stat(OverrideShortnamesConfig); err == nil {
		configs = append(configs, OverrideShortnamesConfig)
	}
	if unshare.IsRootless() {
		path, err := rootlessConfigPath()
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); err == nil {
			configs = append(configs, path)
		}
	}
	return configs, nil
}

func customConfigFile() (string, error) {
	path := os.Getenv("SHORTNAMES_CONF")
	if path != "" {
		return path, nil
	}
	if unshare.IsRootless() {
		path, err := rootlessConfigPath()
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return UserOverrideShortnamesConfig, nil
}

// Writes the configuration to a file
func (conf *Config) writeToFile(path string) error {
	var err error
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	configFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrapf(err, "cannot open %s", path)
	}
	defer configFile.Close()
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(conf); err != nil {
		return err
	}
	buf.WriteTo(configFile)
	if err != nil {
		panic(err)
	}
	return nil
}

// A wrapper for sysregistriesv2
func getRegistries() ([]string, error) {
	registries, err := sysregistriesv2.GetRegistries(nil)
	if err != nil {
		return nil, err
	}
	var registryStrings []string
	for _, registry := range registries {
		registryStrings = append(registryStrings, registry.Prefix)
	}
	return registryStrings, nil
}
