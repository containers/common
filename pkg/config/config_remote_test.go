//go:build remote
// +build remote

package config

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Config Remote", func() {
	BeforeEach(beforeEach)

	It("should succeed on invalid CNIPluginDirs", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		sut.Network.NetworkConfigDir = validDirPath
		sut.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid device mode", func() {
		// Given
		sut.Containers.Devices = []string{"/dev/null:/dev/null:abc"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid first device", func() {
		// Given
		sut.Containers.Devices = []string{"wrong:/dev/null:rw"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid second device", func() {
		// Given
		sut.Containers.Devices = []string{"/dev/null:wrong:rw"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid device", func() {
		// Given
		sut.Containers.Devices = []string{invalidPath}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on wrong invalid device specification", func() {
		// Given
		sut.Containers.Devices = []string{"::::"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("Expect Remote to be true", func() {
		// Given
		// When
		config, err := NewConfig("")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Remote).To(gomega.BeTrue())
	})

	It("should succeed on wrong DefaultUlimits", func() {
		// Given
		sut.Containers.DefaultUlimits = []string{invalidPath}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid CNIPluginDirs", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		sut.Network.NetworkConfigDir = validDirPath
		sut.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed in validating invalid PluginDir", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		sut.Network.NetworkConfigDir = validDirPath
		sut.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})
})
