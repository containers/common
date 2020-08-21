package config

import (
	"os"
)

func customConfigFile() (string, error) {
	path := os.Getenv("CONTAINERS_CONF")
	if path != "" {
		return path, nil
	}
	return rootlessConfigPath()
}
