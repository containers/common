// +build !remote

package config

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = Describe("Config Local", func() {
	BeforeEach(beforeEach)

	It("should fail on invalid NetworkConfigDir", func() {
		// Given
		tmpfile := path.Join(os.TempDir(), "wrong-file")
		file, err := os.Create(tmpfile)
		gomega.Expect(err).To(gomega.BeNil())
		file.Close()
		defer os.Remove(tmpfile)
		sut.Network.NetworkConfigDir = tmpfile
		sut.Network.CNIPluginDirs = []string{}

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid CNIPluginDirs", func() {
		validDirPath, err := ioutil.TempDir("", "config-empty")
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
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail in validating invalid PluginDir", func() {
		validDirPath, err := ioutil.TempDir("", "config-empty")
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
		gomega.Expect(err).ToNot(gomega.BeNil())
	})

	It("should fail on invalid CNIPluginDirs", func() {
		validDirPath, err := ioutil.TempDir("", "config-empty")
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
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail during runtime", func() {
		validDirPath, err := ioutil.TempDir("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		sut.Network.NetworkConfigDir = validDirPath
		tmpDir := path.Join(os.TempDir(), "cni-test")
		sut.Network.CNIPluginDirs = []string{tmpDir}
		defer os.RemoveAll(tmpDir)

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).ToNot(gomega.BeNil())
	})

	It("should fail on invalid device mode", func() {
		// Given
		sut.Containers.Devices = []string{"/dev/null:/dev/null:abc"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid first device", func() {
		// Given
		sut.Containers.Devices = []string{"wrong:/dev/null:rw"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid second device", func() {
		// Given
		sut.Containers.Devices = []string{"/dev/null:wrong:rw"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid device", func() {
		// Given
		sut.Containers.Devices = []string{invalidPath}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on wrong invalid device specification", func() {
		// Given
		sut.Containers.Devices = []string{"::::"}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on wrong DefaultUlimits", func() {
		// Given
		sut.Containers.DefaultUlimits = []string{invalidPath}

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("Expect Remote to be False", func() {
		// Given
		// When
		config, err := NewConfig("")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Remote).To(gomega.BeFalse())
	})

	It("write", func() {
		tmpfile := "containers.conf.test"
		oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
		os.Setenv("CONTAINERS_CONF", tmpfile)
		config, err := ReadCustomConfig()
		gomega.Expect(err).To(gomega.BeNil())
		config.Containers.Devices = []string{"/dev/null:/dev/null:rw",
			"/dev/sdc/",
			"/dev/sdc:/dev/xvdc",
			"/dev/sdc:rm",
		}

		err = config.Write()
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		defer os.Remove(tmpfile)
		// Undo that
		if envSet {
			os.Setenv("CONTAINERS_CONF", oldContainersConf)
		} else {
			os.Unsetenv("CONTAINERS_CONF")
		}
	})

})
