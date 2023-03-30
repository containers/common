//go:build linux
// +build linux

package netavark_test

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/containers/common/libnetwork/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

const pluginName = "netavark-testplugin"

var _ = Describe("Plugins", func() {
	var (
		libpodNet      types.ContainerNetwork
		networkConfDir string
		logBuffer      bytes.Buffer
	)

	BeforeEach(func() {
		var err error
		networkConfDir, err = os.MkdirTemp("", "podman_netavark_test")
		if err != nil {
			Fail("Failed to create tmpdir")
		}
		logBuffer = bytes.Buffer{}
		logrus.SetOutput(&logBuffer)
	})

	JustBeforeEach(func() {
		var err error
		libpodNet, err = getNetworkInterfaceWithPlugins(networkConfDir, []string{"../../bin"})
		if err != nil {
			Fail("Failed to create NewNetworkInterface")
		}
	})

	AfterEach(func() {
		os.RemoveAll(networkConfDir)
	})

	It("create plugin network", func() {
		network := types.Network{Driver: pluginName}
		network1, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(BeNil())
		Expect(network1.Name).ToNot(BeEmpty())
		Expect(network1.ID).ToNot(BeEmpty())
		Expect(filepath.Join(networkConfDir, network1.Name+".json")).To(BeARegularFile())
	})

	It("create plugin network with name", func() {
		name := "test123"
		network := types.Network{Driver: pluginName, Name: name}
		network1, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(BeNil())
		Expect(network1.Name).To(Equal(name))
		Expect(network1.ID).ToNot(BeEmpty())
		Expect(filepath.Join(networkConfDir, network1.Name+".json")).To(BeARegularFile())
	})

	It("create plugin error", func() {
		network := types.Network{
			Driver:  pluginName,
			Options: map[string]string{"error": "my custom error"},
		}
		_, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("plugin ../../bin/netavark-testplugin failed: netavark (exit code 1): my custom error"))
	})

	It("create plugin change name error", func() {
		network := types.Network{
			Driver:  pluginName,
			Options: map[string]string{"name": "newName"},
		}
		_, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid plugin result: changed network name"))
	})

	It("create plugin change id error", func() {
		network := types.Network{
			Driver:  pluginName,
			Options: map[string]string{"id": "newID"},
		}
		_, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid plugin result: changed network ID"))
	})

	It("create plugin change driver error", func() {
		network := types.Network{
			Driver:  pluginName,
			Options: map[string]string{"driver": "newDriver"},
		}
		_, err := libpodNet.NetworkCreate(network, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid plugin result: changed network driver"))
	})
})
