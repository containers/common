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
			defaultConfig, err := config.NewConfig("")

			// Then
			Expect(err).To(BeNil())
			Expect(defaultConfig.ApparmorProfile).To(Equal("container-default"))
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
			sut.NetworkConfig.NetworkConfigDir = validDirPath
			tmpDir := path.Join(os.TempDir(), "cni-test")
			sut.NetworkConfig.CNIPluginDirs = []string{tmpDir}
			defer os.RemoveAll(tmpDir)

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
		})

		It("should create the  NetworkConfigDir", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			tmpDir := path.Join(os.TempDir(), invalidPath)
			sut.NetworkConfig.NetworkConfigDir = tmpDir
			sut.NetworkConfig.CNIPluginDirs = []string{validDirPath}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).To(BeNil())
			os.RemoveAll(tmpDir)
		})

		It("should fail on invalid NetworkConfigDir", func() {
			// Given
			tmpfile := path.Join(os.TempDir(), "wrong-file")
			file, err := os.Create(tmpfile)
			Expect(err).To(BeNil())
			file.Close()
			defer os.Remove(tmpfile)
			sut.NetworkConfig.NetworkConfigDir = tmpfile
			sut.NetworkConfig.CNIPluginDirs = []string{}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid CNIPluginDirs", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkConfigDir = validDirPath
			sut.NetworkConfig.CNIPluginDirs = []string{invalidPath}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail in validating invalid PluginDir", func() {
			validDirPath, err := ioutil.TempDir("", "config-empty")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(validDirPath)
			// Given
			sut.NetworkConfig.NetworkConfigDir = validDirPath
			sut.NetworkConfig.CNIPluginDirs = []string{invalidPath}

			// When
			err = sut.NetworkConfig.Validate(true)

			// Then
			Expect(err).ToNot(BeNil())
		})

	})

	Describe("ReadConfigFromFile", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, err := config.ReadConfigFromFile("testdata/containers_default.conf")

			OCIRuntimeMap := map[string][]string{
				"runc": []string{
					"/usr/bin/runc",
					"/usr/sbin/runc",
					"/usr/local/bin/runc",
					"/usr/local/sbin/runc",
					"/sbin/runc",
					"/bin/runc",
					"/usr/lib/cri-o-runc/sbin/runc",
				},
				"crun": []string{
					"/usr/bin/crun",
					"/usr/local/bin/crun",
				},
			}

			pluginDirs := []string{
				"/usr/libexec/cni",
				"/usr/lib/cni",
				"/usr/local/lib/cni",
				"/opt/cni/bin",
			}

			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// Then
			Expect(err).To(BeNil())
			Expect(defaultConfig.CgroupManager).To(Equal("systemd"))
			Expect(defaultConfig.Env).To(BeEquivalentTo(envs))
			Expect(defaultConfig.PidsLimit).To(BeEquivalentTo(2048))
			Expect(defaultConfig.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(defaultConfig.NumLocks).To(BeEquivalentTo(2048))
			Expect(defaultConfig.OCIRuntimes).To(Equal(OCIRuntimeMap))
		})

		It("should succeed with commented out configuration", func() {
			// Given
			// When
			_, err := config.ReadConfigFromFile("testdata/containers_comment.conf")

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail when file does not exist", func() {
			// Given
			// When
			_, err := config.ReadConfigFromFile("/invalid/file")

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			// When
			_, err := config.ReadConfigFromFile("config.go")

			// Then
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("NewConfig", func() {
		It("should success with default config", func() {
			// Given
			OCIRuntimeMap := map[string][]string{
				"runc": []string{
					"/usr/bin/runc",
					"/usr/sbin/runc",
					"/usr/local/bin/runc",
					"/usr/local/sbin/runc",
					"/sbin/runc",
					"/bin/runc",
					"/usr/lib/cri-o-runc/sbin/runc",
					"/run/current-system/sw/bin/runc",
				},
				"crun": []string{
					"/usr/bin/crun",
					"/usr/sbin/crun",
					"/usr/local/bin/crun",
					"/usr/local/sbin/crun",
					"/sbin/crun",
					"/bin/crun",
					"/run/current-system/sw/bin/crun",
				},
			}

			pluginDirs := []string{
				"/usr/libexec/cni",
				"/usr/lib/cni",
				"/usr/local/lib/cni",
				"/opt/cni/bin",
			}

			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// When
			config, err := config.NewConfig("")
			// Then
			Expect(err).To(BeNil())
			Expect(config.ApparmorProfile).To(Equal("container-default"))
			Expect(config.PidsLimit).To(BeEquivalentTo(2048))
			Expect(config.Env).To(BeEquivalentTo(envs))
			Expect(config.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(config.NumLocks).To(BeEquivalentTo(2048))
			Expect(config.OCIRuntimes["runc"]).To(Equal(OCIRuntimeMap["runc"]))
		})

		It("should success with valid user file path", func() {
			// Given
			// When
			config, err := config.NewConfig("testdata/containers_default.conf")
			// Then
			Expect(err).To(BeNil())
			Expect(config.ApparmorProfile).To(Equal("container-default"))
			Expect(config.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("should fail with invalid value", func() {
			// Given
			// When
			config, err := config.NewConfig("testdata/containers_invalid.conf")
			// Then
			Expect(err).ToNot(BeNil())
			Expect(config).To(BeNil())
		})
	})
})
