package config

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	BeforeEach(beforeEach)

	Describe("ValidateConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, err := NewConfig("")

			// Then
			Expect(err).To(BeNil())
			Expect(defaultConfig.Containers.ApparmorProfile).To(Equal("container-default"))
			Expect(defaultConfig.Containers.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("should succeed with additional devices", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{"/dev/null:/dev/null:rw",
				"/dev/sdc/",
				"/dev/sdc:/dev/xvdc",
				"/dev/sdc:rm",
			}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail on wrong DefaultUlimits", func() {
			// Given
			sut.Containers.DefaultUlimits = []string{invalidPath}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on wrong invalid device specification", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{"::::"}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid device", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{invalidPath}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid device mode", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{"/dev/null:/dev/null:abc"}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid first device", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{"wrong:/dev/null:rw"}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail on invalid second device", func() {
			// Given
			sut.Containers.AdditionalDevices = []string{"/dev/null:wrong:rw"}

			// When
			err := sut.Containers.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail wrong max log size", func() {
			// Given
			sut.Containers.LogSizeMax = 1

			// When
			err := sut.Validate(true)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should succeed with valid shm size", func() {
			// Given
			sut.Containers.ShmSize = "1024"

			// When
			err := sut.Validate(true)

			// Then
			Expect(err).To(BeNil())

			// Given
			sut.Containers.ShmSize = "64m"

			// When
			err = sut.Validate(true)

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail wrong shm size", func() {
			// Given
			sut.Containers.ShmSize = "-2"

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
			err := sut.Network.Validate(false)

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
			sut.Network.NetworkConfigDir = validDirPath
			tmpDir := path.Join(os.TempDir(), "cni-test")
			sut.Network.CNIPluginDirs = []string{tmpDir}
			defer os.RemoveAll(tmpDir)

			// When
			err = sut.Network.Validate(true)

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
			sut.Network.NetworkConfigDir = tmpDir
			sut.Network.CNIPluginDirs = []string{validDirPath}

			// When
			err = sut.Network.Validate(true)

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
			sut.Network.NetworkConfigDir = tmpfile
			sut.Network.CNIPluginDirs = []string{}

			// When
			err = sut.Network.Validate(true)

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
			sut.Network.NetworkConfigDir = validDirPath
			sut.Network.CNIPluginDirs = []string{invalidPath}

			// When
			err = sut.Network.Validate(true)

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
			sut.Network.NetworkConfigDir = validDirPath
			sut.Network.CNIPluginDirs = []string{invalidPath}

			// When
			err = sut.Network.Validate(true)

			// Then
			Expect(err).ToNot(BeNil())
		})

	})

	Describe("ReadConfigFromFile", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			conf, _ := DefaultConfig()
			defaultConfig, err := ReadConfigFromFile("testdata/containers_default.conf", conf)

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
			Expect(defaultConfig.Containers.CgroupManager).To(Equal("systemd"))
			Expect(defaultConfig.Containers.Env).To(BeEquivalentTo(envs))
			Expect(defaultConfig.Containers.PidsLimit).To(BeEquivalentTo(2048))
			Expect(defaultConfig.Network.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(defaultConfig.Libpod.NumLocks).To(BeEquivalentTo(2048))
			Expect(defaultConfig.Libpod.OCIRuntimes).To(Equal(OCIRuntimeMap))
		})

		It("should succeed with commented out configuration", func() {
			// Given
			// When
			conf := Config{}
			_, err := ReadConfigFromFile("testdata/containers_comment.conf", &conf)

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail when file does not exist", func() {
			// Given
			// When
			conf := Config{}
			_, err := ReadConfigFromFile("/invalid/file", &conf)

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			// When
			conf := Config{}
			_, err := ReadConfigFromFile("config.go", &conf)

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
			config, err := NewConfig("")
			// Then
			Expect(err).To(BeNil())
			Expect(config.Containers.ApparmorProfile).To(Equal("container-default"))
			Expect(config.Containers.PidsLimit).To(BeEquivalentTo(2048))
			Expect(config.Containers.Env).To(BeEquivalentTo(envs))
			Expect(config.Network.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(config.Libpod.NumLocks).To(BeEquivalentTo(2048))
			Expect(config.Libpod.OCIRuntimes["runc"]).To(Equal(OCIRuntimeMap["runc"]))
		})

		It("verify getDefaultEnv", func() {
			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// When
			config, err := NewConfig("")
			// Then
			Expect(err).To(BeNil())
			Expect(config.GetDefaultEnv()).To(BeEquivalentTo(envs))
			config.Containers.HTTPProxy = true
			Expect(config.GetDefaultEnv()).To(BeEquivalentTo(envs))
			os.Setenv("HTTP_PROXY", "localhost")
			os.Setenv("FOO", "BAR")
			newenvs := []string{"HTTP_PROXY=localhost"}
			envs = append(newenvs, envs...)
			Expect(config.GetDefaultEnv()).To(BeEquivalentTo(envs))
			config.Containers.HTTPProxy = false
			config.Containers.EnvHost = true
			envString := strings.Join(config.GetDefaultEnv(), ",")
			Expect(envString).To(ContainSubstring("FOO=BAR"))
			Expect(envString).To(ContainSubstring("HTTP_PROXY=localhost"))
		})

		It("should success with valid user file path", func() {
			// Given
			// When
			config, err := NewConfig("testdata/containers_default.conf")
			// Then
			Expect(err).To(BeNil())
			Expect(config.Containers.ApparmorProfile).To(Equal("container-default"))
			Expect(config.Containers.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("should fail with invalid value", func() {
			// Given
			// When
			config, err := NewConfig("testdata/containers_invalid.conf")
			// Then
			Expect(err).ToNot(BeNil())
			Expect(config).To(BeNil())
		})
	})
})
