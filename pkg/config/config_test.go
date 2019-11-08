package config_test

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/containers/common/pkg/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	BeforeEach(beforeEach)

	Describe("ValidateConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, err := config.New("")

			// Then
			Expect(err).To(BeNil())
			Expect(defaultConfig.CgroupManager).To(Equal("systemd"))
			Expect(defaultConfig.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("should succeed with additional devices", func() {
			// Given
			sut.AdditionalDevices = []string{"/dev/null:/dev/null:rw",
				"/dev/sdc/",
				"/dev/sdc:/dev/xvdc",
				"/dev/sdc:rm",
			}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail on wrong DefaultUlimits", func() {
			// Given
			sut.DefaultUlimits = []string{invalidPath}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on wrong invalid device specification", func() {
			// Given
			sut.AdditionalDevices = []string{"::::"}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid device", func() {
			// Given
			sut.AdditionalDevices = []string{invalidPath}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid device mode", func() {
			// Given
			sut.AdditionalDevices = []string{"/dev/null:/dev/null:abc"}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid first device", func() {
			// Given
			sut.AdditionalDevices = []string{"wrong:/dev/null:rw"}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid second device", func() {
			// Given
			sut.AdditionalDevices = []string{"/dev/null:wrong:rw"}

			// When
			err := sut.ContainersConfig.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail wrong max log size", func() {
			// Given
			sut.LogSizeMax = 1

			// When
			err := sut.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should succeed with valid shm size", func() {
			// Given
			sut.ShmSize = "1024"

			// When
			err := sut.Validate(true)

			// Then
			Expect(err).To(BeNil())

			// Given
			sut.ShmSize = "64m"

			// When
			err = sut.Validate(true)

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail wrong shm size", func() {
			// Given
			sut.ShmSize = "-2"

			// When
			err := sut.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

	})

	Describe("ValidateNetworkConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			err := sut.NetworkConfig.Validate(false)

			// Then
			Expect(err).To(BeNil())
		})

		It("should succeed during runtime", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkDir = validDirPath
			tmpDir := path.Join(os.TempDir(), "cni-test")
			sut.NetworkConfig.PluginDirs = []string{tmpDir}
			defer os.RemoveAll(tmpDir)

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
		})

		It("should create the  NetworkDir", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			tmpDir := path.Join(os.TempDir(), invalidPath)
			sut.NetworkConfig.NetworkDir = tmpDir
			sut.NetworkConfig.PluginDirs = []string{validDirPath}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
			os.RemoveAll(tmpDir)
		})

		It("should fail on invalid NetworkDir", func() {
			// Given
			tmpfile := path.Join(os.TempDir(), "wrong-file")
			file, err := os.Create(tmpfile)
			Expect(err).To(BeNil())
			file.Close()
			defer os.Remove(tmpfile)
			sut.NetworkConfig.NetworkDir = tmpfile
			sut.NetworkConfig.PluginDirs = []string{}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid PluginDirs", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkDir = validDirPath
			sut.NetworkConfig.PluginDirs = []string{invalidPath}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should succeed on having PluginDir", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkDir = validDirPath
			sut.NetworkConfig.PluginDir = validDirPath
			sut.NetworkConfig.PluginDirs = []string{}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
		})

		It("should succeed in appending PluginDir to PluginDirs", func() {

			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)

			// Given
			sut.NetworkConfig.NetworkDir = validDirPath
			sut.NetworkConfig.PluginDir = validDirPath
			sut.NetworkConfig.PluginDirs = []string{}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
			Expect(sut.NetworkConfig.PluginDirs[0]).To(Equal(validDirPath))
		})

		It("should fail in validating invalid PluginDir", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkDir = validDirPath
			sut.NetworkConfig.PluginDir = invalidPath
			sut.NetworkConfig.PluginDirs = []string{}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).ToNot(BeNil())
		})

	})

	Describe("UpdateFromFile", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			err := sut.UpdateFromFile("testdata/containers_default.conf")

			// Then
			Expect(err).To(BeNil())
			Expect(sut.CgroupManager).To(Equal("systemd"))
			Expect(sut.PidsLimit).To(BeEquivalentTo(1024))
		})

		It("should succeed with commented out configuration", func() {
			// Given
			// When
			err := sut.UpdateFromFile("testdata/containers_comment.conf")

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail when file does not exist", func() {
			// Given
			// When
			err := sut.UpdateFromFile("/invalid/file")

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			// When
			err := sut.UpdateFromFile("config.go")

			// Then
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("New", func() {
		It("should success with default config", func() {
			// Given
			// When
			config, err := config.New("")
			// Then
			Expect(err).To(BeNil())
			Expect(config.CgroupManager).To(Equal("systemd"))
			Expect(config.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("should success with valid user file path", func() {
			// Given
			// When
			config, err := config.New("testdata/containers_default.conf")
			// Then
			Expect(err).To(BeNil())
			Expect(config.CgroupManager).To(Equal("systemd"))
			Expect(config.PidsLimit).To(BeEquivalentTo(1024))
		})

		It("should fail with invalid value", func() {
			// Given
			// When
			config, err := config.New("testdata/containers_invalid.conf")
			// Then
			Expect(err).ToNot(BeNil())
			Expect(config).To(BeNil())
		})
	})
})
