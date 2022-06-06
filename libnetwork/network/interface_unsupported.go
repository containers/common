//go:build !linux && !freebsd
// +build !linux,!freebsd

package network

import (
	"fmt"

	"github.com/containers/common/libnetwork/dummy"
	"github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
)

func NetworkBackend(store storage.Store, conf *config.Config, syslog bool) (types.NetworkBackend, types.ContainerNetwork, error) {
	backend := types.NetworkBackend(conf.Network.NetworkBackend)
	if backend == "" {
		var err error
		backend, err = defaultNetworkBackend(store, conf)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get default network backend: %w", err)
		}
	}

	switch backend {
	case types.Dummy:
		netInt := dummy.NewDummyNetworkInterface()
		return types.Dummy, netInt, nil

	default:
		return "", nil, fmt.Errorf("unsupported network backend %q, check network_backend in containers.conf", backend)
	}
}

func defaultNetworkBackend(store storage.Store, conf *config.Config) (backend types.NetworkBackend, err error) {
	return types.Dummy, nil
}
