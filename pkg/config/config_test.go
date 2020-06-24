package config

import (
	"os"
	"sort"
	"strings"

	"github.com/containers/common/pkg/apparmor"
	"github.com/containers/common/pkg/capabilities"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	selinux "github.com/opencontainers/selinux/go-selinux"
)

var _ = Describe("Config", func() {
	BeforeEach(beforeEach)

	Describe("ValidateConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, err := NewConfig("")

			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defaultConfig.Containers.ApparmorProfile).To(gomega.Equal(apparmor.Profile))
			gomega.Expect(defaultConfig.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
		})

		It("should succeed with devices", func() {
			// Given
			sut.Containers.Devices = []string{"/dev/null:/dev/null:rw",
				"/dev/sdc/",
				"/dev/sdc:/dev/xvdc",
				"/dev/sdc:rm",
			}

			// When
			err := sut.Containers.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail wrong max log size", func() {
			// Given
			sut.Containers.LogSizeMax = 1

			// When
			err := sut.Validate()

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("should succeed with valid shm size", func() {
			// Given
			sut.Containers.ShmSize = "1024"

			// When
			err := sut.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())

			// Given
			sut.Containers.ShmSize = "64m"

			// When
			err = sut.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail wrong shm size", func() {
			// Given
			sut.Containers.ShmSize = "-2"

			// When
			err := sut.Validate()

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("Check SELinux settings", func() {
			if selinux.GetEnabled() {
				sut.Containers.EnableLabeling = true
				gomega.Expect(sut.Containers.Validate()).To(gomega.BeNil())
				gomega.Expect(selinux.GetEnabled()).To(gomega.BeTrue())

				sut.Containers.EnableLabeling = false
				gomega.Expect(sut.Containers.Validate()).To(gomega.BeNil())
				gomega.Expect(selinux.GetEnabled()).To(gomega.BeFalse())
			}

		})

	})

	Describe("ValidateNetworkConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			err := sut.Network.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})
	})

	Describe("readConfigFromFile", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			conf, _ := DefaultConfig()
			defaultConfig, err := readConfigFromFile("testdata/containers_default.conf", conf)

			OCIRuntimeMap := map[string][]string{
				"kata": {

					"/usr/bin/kata-runtime",
					"/usr/sbin/kata-runtime",
					"/usr/local/bin/kata-runtime",
					"/usr/local/sbin/kata-runtime",
					"/sbin/kata-runtime",
					"/bin/kata-runtime",
					"/usr/bin/kata-qemu",
					"/usr/bin/kata-fc",
				},
				"runc": {
					"/usr/bin/runc",
					"/usr/sbin/runc",
					"/usr/local/bin/runc",
					"/usr/local/sbin/runc",
					"/sbin/runc",
					"/bin/runc",
					"/usr/lib/cri-o-runc/sbin/runc",
				},
				"crun": {
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
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defaultConfig.Engine.CgroupManager).To(gomega.Equal("systemd"))
			gomega.Expect(defaultConfig.Containers.Env).To(gomega.BeEquivalentTo(envs))
			gomega.Expect(defaultConfig.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(defaultConfig.Network.CNIPluginDirs).To(gomega.Equal(pluginDirs))
			gomega.Expect(defaultConfig.Engine.NumLocks).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(defaultConfig.Engine.OCIRuntimes).To(gomega.Equal(OCIRuntimeMap))
		})

		It("should succeed with commented out configuration", func() {
			// Given
			// When
			conf := Config{}
			_, err := readConfigFromFile("testdata/containers_comment.conf", &conf)

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail when file does not exist", func() {
			// Given
			// When
			conf := Config{}
			_, err := readConfigFromFile("/invalid/file", &conf)

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			// When
			conf := Config{}
			_, err := readConfigFromFile("config.go", &conf)

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})
	})

	Describe("NewConfig", func() {
		It("should success with default config", func() {
			// Given
			OCIRuntimeMap := map[string][]string{
				"runc": {
					"/usr/bin/runc",
					"/usr/sbin/runc",
					"/usr/local/bin/runc",
					"/usr/local/sbin/runc",
					"/sbin/runc",
					"/bin/runc",
					"/usr/lib/cri-o-runc/sbin/runc",
					"/run/current-system/sw/bin/runc",
				},
				"crun": {
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
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal(apparmor.Profile))
			gomega.Expect(config.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(config.Containers.Env).To(gomega.BeEquivalentTo(envs))
			gomega.Expect(config.Network.CNIPluginDirs).To(gomega.Equal(pluginDirs))
			gomega.Expect(config.Engine.NumLocks).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(config.Engine.OCIRuntimes["runc"]).To(gomega.Equal(OCIRuntimeMap["runc"]))
		})

		It("verify getDefaultEnv", func() {
			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// When
			config, err := Default()
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config.GetDefaultEnv()).To(gomega.BeEquivalentTo(envs))
			config.Containers.HTTPProxy = true
			gomega.Expect(config.GetDefaultEnv()).To(gomega.BeEquivalentTo(envs))
			os.Setenv("HTTP_PROXY", "localhost")
			os.Setenv("FOO", "BAR")
			newenvs := []string{"HTTP_PROXY=localhost"}
			envs = append(newenvs, envs...)
			gomega.Expect(config.GetDefaultEnv()).To(gomega.BeEquivalentTo(envs))
			config.Containers.HTTPProxy = false
			config.Containers.EnvHost = true
			envString := strings.Join(config.GetDefaultEnv(), ",")
			gomega.Expect(envString).To(gomega.ContainSubstring("FOO=BAR"))
			gomega.Expect(envString).To(gomega.ContainSubstring("HTTP_PROXY=localhost"))
		})

		It("should success with valid user file path", func() {
			// Given
			// When
			config, err := NewConfig("testdata/containers_default.conf")
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("container-default"))
			gomega.Expect(config.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
		})

		It("contents of passed-in file should override others", func() {
			// Given we do
			oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
			os.Setenv("CONTAINERS_CONF", "containers.conf")
			// When
			config, err := NewConfig("testdata/containers_override.conf")
			// Undo that
			if envSet {
				os.Setenv("CONTAINERS_CONF", oldContainersConf)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config).ToNot(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("overridden-default"))
		})

		It("should fail with invalid value", func() {
			// Given
			// When
			config, err := NewConfig("testdata/containers_invalid.conf")
			// Then
			gomega.Expect(err).ToNot(gomega.BeNil())
			gomega.Expect(config).To(gomega.BeNil())
		})

		It("Test Capabilities call", func() {
			// Given
			// When
			config, err := NewConfig("")
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			var addcaps, dropcaps []string
			caps, err := config.Capabilities("0", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			defaultCaps := config.Containers.DefaultCapabilities
			sort.Strings(defaultCaps)
			gomega.Expect(caps).To(gomega.BeEquivalentTo(defaultCaps))

			// Add all caps
			addcaps = []string{"all"}
			caps, err = config.Capabilities("root", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			gomega.Expect(caps).To(gomega.BeEquivalentTo(capabilities.AllCapabilities()))

			// Drop all caps
			dropcaps = []string{"all"}
			caps, err = config.Capabilities("", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			gomega.Expect(caps).ToNot(gomega.BeEquivalentTo([]string{}))

			config.Containers.DefaultCapabilities = []string{
				"CAP_AUDIT_WRITE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FOWNER",
			}

			expectedCaps := []string{
				"CAP_AUDIT_WRITE",
				"CAP_DAC_OVERRIDE",
				"CAP_NET_ADMIN",
				"CAP_SYS_ADMIN",
			}

			// Add all caps
			addcaps = []string{"CAP_NET_ADMIN", "CAP_SYS_ADMIN"}
			dropcaps = []string{"CAP_FOWNER", "CAP_CHOWN"}
			caps, err = config.Capabilities("", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			gomega.Expect(caps).To(gomega.BeEquivalentTo(expectedCaps))

			addcaps = []string{"NET_ADMIN", "cap_sys_admin"}
			dropcaps = []string{"FOWNER", "chown"}
			caps, err = config.Capabilities("", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			gomega.Expect(caps).To(gomega.BeEquivalentTo(expectedCaps))

			expectedCaps = []string{"CAP_NET_ADMIN", "CAP_SYS_ADMIN"}
			caps, err = config.Capabilities("notroot", addcaps, dropcaps)
			gomega.Expect(err).To(gomega.BeNil())
			sort.Strings(caps)
			gomega.Expect(caps).To(gomega.BeEquivalentTo(expectedCaps))
		})

		It("should succeed with default pull_policy", func() {
			err := sut.Engine.Validate()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(sut.Engine.PullPolicy).To(gomega.Equal("missing"))

			sut.Engine.PullPolicy = DefaultPullPolicy
			err = sut.Engine.Validate()
			gomega.Expect(err).To(gomega.BeNil())
		})
		It("should fail with invalid pull_policy", func() {
			sut.Engine.PullPolicy = "invalidPullPolicy"
			err := sut.Engine.Validate()
			gomega.Expect(err).ToNot(gomega.BeNil())
		})
	})

})
