//go:build remote
// +build remote

package config

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Config Remote", func() {
	It("should succeed on invalid CNIPluginDirs", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		defConf.Network.NetworkConfigDir = validDirPath
		defConf.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid device mode", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.Devices = []string{"/dev/null:/dev/null:abc"}

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid first device", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.Devices = []string{"wrong:/dev/null:rw"}

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid second device", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.Devices = []string{"/dev/null:wrong:rw"}

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on invalid device", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.Devices = []string{invalidPath}

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on wrong invalid device specification", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.Devices = []string{"::::"}

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("Expect Remote to be true", func() {
		// Given
		// When
		config, err := New(nil)
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Remote).To(gomega.BeTrue())
	})

	It("should succeed on wrong DefaultUlimits", func() {
		// Given
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Containers.DefaultUlimits = []string{invalidPath}

		// When
		err = defConf.Containers.Validate()

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
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Network.NetworkConfigDir = validDirPath
		defConf.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = defConf.Network.Validate()

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
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())
		defConf.Network.NetworkConfigDir = validDirPath
		defConf.Network.CNIPluginDirs = []string{invalidPath}

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})
})
