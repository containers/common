package config

import (
	"os"

	"github.com/containers/storage/pkg/unshare"
	selinux "github.com/opencontainers/selinux/go-selinux"
)

func selinuxEnabled() bool {
	return selinux.GetEnabled()
}

func customConfigFile() (string, error) {
	path := os.Getenv("CONTAINERS_CONF")
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
	return OverrideContainersConfig, nil
}
