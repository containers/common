package machine

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Machine", func() {
	BeforeEach(func() {
		// disable normal init for testing
		markerSync.Do(func() {})
	})

	It("not a machine", func() {
		loadMachineMarker("testdata/does-not-exist")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeFalse())
		gomega.Expect(HostType()).To(gomega.BeEmpty())
		gomega.Expect(IsGvProxyBased()).To(gomega.BeFalse())
	})

	It("generic machine", func() {
		loadMachineMarker("testdata/empty-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(HostType()).To(gomega.BeEmpty())
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})

	It("wsl machine", func() {
		loadMachineMarker("testdata/wsl-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(HostType()).To(gomega.Equal(Wsl))
		gomega.Expect(IsGvProxyBased()).To(gomega.BeFalse())
	})

	It("qemu machine", func() {
		loadMachineMarker("testdata/qemu-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(HostType()).To(gomega.Equal(Qemu))
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})

	It("applehv machine", func() {
		loadMachineMarker("testdata/applehv-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(HostType()).To(gomega.Equal(AppleHV))
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})

	It("hyperv machine", func() {
		loadMachineMarker("testdata/hyperv-machine")

		gomega.Expect(IsPodmanMachine()).To(gomega.BeTrue())
		gomega.Expect(HostType()).To(gomega.Equal(HyperV))
		gomega.Expect(IsGvProxyBased()).To(gomega.BeTrue())
	})
})
