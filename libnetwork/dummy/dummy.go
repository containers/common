package dummy

import (
	"github.com/containers/common/libnetwork/types"
	"github.com/pkg/errors"
)

type dummyNetwork struct{}

// NetworkCreate will take a partial filled Network and fill the
// missing fields. It creates the Network and returns the full Network.
func (n *dummyNetwork) NetworkCreate(net types.Network) (types.Network, error) {
	return types.Network{}, errors.New("function not supported by the dummy network backend")
}

// NetworkRemove will remove the Network with the given name or ID.
// It does not ensure that the network is unused.
func (n *dummyNetwork) NetworkRemove(nameOrID string) error {
	return errors.New("function not supported by the dummy network backend")
}

// NetworkList will return all known Networks. Optionally you can
// supply a list of filter functions. Only if a network matches all
// functions it is returned.
func (n *dummyNetwork) NetworkList(filters ...types.FilterFunc) ([]types.Network, error) {
	return nil, errors.New("function not supported by the dummy network backend")
}

// NetworkInspect will return the Network with the given name or ID.
func (n *dummyNetwork) NetworkInspect(nameOrID string) (types.Network, error) {
	return types.Network{}, errors.New("function not supported by the dummy network backend")
}

// Setup will setup the container network namespace. It returns
// a map of StatusBlocks, the key is the network name.
func (n *dummyNetwork) Setup(namespacePath string, options types.SetupOptions) (map[string]types.StatusBlock, error) {
	return nil, errors.New("function not supported by the dummy network backend")
}

// Teardown will teardown the container network namespace.
func (n *dummyNetwork) Teardown(namespacePath string, options types.TeardownOptions) error {
	return errors.New("function not supported by the dummy network backend")
}

// Drivers will return the list of supported network drivers
// for this interface.
func (n *dummyNetwork) Drivers() []string {
	return []string{}
}

// DefaultNetworkName will return the default cni network name.
func (n *dummyNetwork) DefaultNetworkName() string {
	return "dummy"
}

// NewNetworkInterface creates the ContainerNetwork interface for the dummy backend.
func NewDummyNetworkInterface() types.ContainerNetwork {
	return &dummyNetwork{}
}
