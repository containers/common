package machine

import (
	"github.com/containers/common/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Machine", func() {
	BeforeEach(func() {
		// disable normal init for testing
		markerSync.Do(func() {})

		// ensure legacy flag is off
		config, _ := config.Default()
		//nolint:staticcheck //lint:ignore SA1019 deprecated call
		config.Engine.MachineEnabled = false
	})

	It("not a machine", func() {
		loadMachineMarker("testdata/does-not-exist")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeFalse())
		gomega.Expect(MachineHostType()).To(gomega.BeEmpty())
		gomega.Expect(IsGvProxyBased()).To(gomega.BeFalse())
	})

	It("generic machine", func() {
		loadMachineMarker("testdata/empty-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(MachineHostType()).To(gomega.BeEmpty())
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})

	It("wsl machine", func() {
		loadMachineMarker("testdata/wsl-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(MachineHostType()).To(gomega.Equal(Wsl))
		gomega.Expect(IsGvProxyBased()).To(gomega.BeFalse())
	})

	It("legacy config machine", func() {
		config, _ := config.Default()
		//nolint:staticcheck //lint:ignore SA1019 deprecated call
		config.Engine.MachineEnabled = true
		loadMachineMarker("testdata/does-not-exist")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(MachineHostType()).To(gomega.BeEmpty())
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})
})
