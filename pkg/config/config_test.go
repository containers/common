package config_test

import (
	"os"
	"sort"
	"strings"

	"github.com/containers/common/pkg/apparmor"
	"github.com/containers/common/pkg/capabilities"
	"github.com/containers/common/pkg/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	selinux "github.com/opencontainers/selinux/go-selinux"
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
			Expect(defaultConfig.Containers.ApparmorProfile).To(Equal(apparmor.Profile))
			Expect(defaultConfig.Containers.PidsLimit).To(BeEquivalentTo(2048))
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
			Expect(err).To(BeNil())
		})

		It("should fail wrong max log size", func() {
			// Given
			sut.Containers.LogSizeMax = 1

			// When
			err := sut.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("should succeed with valid shm size", func() {
			// Given
			sut.Containers.ShmSize = "1024"

			// When
			err := sut.Validate()

			// Then
			Expect(err).To(BeNil())

			// Given
			sut.Containers.ShmSize = "64m"

			// When
			err = sut.Validate()

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail wrong shm size", func() {
			// Given
			sut.Containers.ShmSize = "-2"

			// When
			err := sut.Validate()

			// Then
			Expect(err).NotTo(BeNil())
		})

		It("Check SELinux settings", func() {
			if selinux.GetEnabled() {
				sut.Containers.EnableLabeling = true
				Expect(sut.Containers.Validate()).To(BeNil())
				Expect(selinux.GetEnabled()).To(BeTrue())

				sut.Containers.EnableLabeling = false
				Expect(sut.Containers.Validate()).To(BeNil())
				Expect(selinux.GetEnabled()).To(BeFalse())
			}

		})

	})

	Describe("ValidateNetworkConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			err := sut.Network.Validate()

			// Then
			Expect(err).To(BeNil())
		})
	})

	Describe("ReadCustomConfig", func() {
		It("should succeed with default config", func() {
			// Given
			os.Setenv("CONTAINERS_CONF", "testdata/containers_default.conf")
			defer os.Unsetenv("CONTAINERS_CONF")

			// When
			defaultConfig, err := config.ReadCustomConfig()

			OCIRuntimeMap := map[string][]string{
				"crun": {
					"/usr/bin/crun",
					"/usr/local/bin/crun",
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
			Expect(defaultConfig.Engine.CgroupManager).To(Equal("systemd"))
			Expect(defaultConfig.Containers.Env).To(BeEquivalentTo(envs))
			Expect(defaultConfig.Containers.PidsLimit).To(BeEquivalentTo(2048))
			Expect(defaultConfig.Network.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(defaultConfig.Engine.NumLocks).To(BeEquivalentTo(2048))
			Expect(defaultConfig.Engine.OCIRuntimes).To(Equal(OCIRuntimeMap))
		})

		It("should succeed with commented out configuration", func() {
			// Given
			os.Setenv("CONTAINERS_CONF", "testdata/containers_comment.conf")
			defer os.Unsetenv("CONTAINERS_CONF")

			// When
			_, err := config.ReadCustomConfig()

			// Then
			Expect(err).To(BeNil())
		})

		It("should succeed when file does not exist", func() {
			// Given
			os.Setenv("CONTAINERS_CONF", "/invalid/file")
			defer os.Unsetenv("CONTAINERS_CONF")

			// When
			_, err := config.ReadCustomConfig()

			// Then
			Expect(err).To(BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			os.Setenv("CONTAINERS_CONF", "config.go")
			defer os.Unsetenv("CONTAINERS_CONF")

			// When
			_, err := config.ReadCustomConfig()

			// Then
			Expect(err).NotTo(BeNil())
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
			config, err := config.NewConfig("")
			// Then
			Expect(err).To(BeNil())
			Expect(config.Containers.ApparmorProfile).To(Equal(apparmor.Profile))
			Expect(config.Containers.PidsLimit).To(BeEquivalentTo(2048))
			Expect(config.Containers.Env).To(BeEquivalentTo(envs))
			Expect(config.Network.CNIPluginDirs).To(Equal(pluginDirs))
			Expect(config.Engine.NumLocks).To(BeEquivalentTo(2048))
			Expect(config.Engine.OCIRuntimes["runc"]).To(Equal(OCIRuntimeMap["runc"]))
		})

		It("verify getDefaultEnv", func() {
			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// When
			config, err := config.Default()
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
			config, err := config.NewConfig("testdata/containers_default.conf")
			// Then
			Expect(err).To(BeNil())
			Expect(config.Containers.ApparmorProfile).To(Equal("container-default"))
			Expect(config.Containers.PidsLimit).To(BeEquivalentTo(2048))
		})

		It("contents of passed-in file should override others", func() {
			// Given we do
			oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
			os.Setenv("CONTAINERS_CONF", "containers.conf")
			// When
			config, err := config.NewConfig("testdata/containers_override.conf")
			// Undo that
			if envSet {
				os.Setenv("CONTAINERS_CONF", oldContainersConf)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
			// Then
			Expect(err).To(BeNil())
			Expect(config).ToNot(BeNil())
			Expect(config.Containers.ApparmorProfile).To(Equal("overridden-default"))
		})

		It("should fail with invalid value", func() {
			// Given
			// When
			config, err := config.NewConfig("testdata/containers_invalid.conf")
			// Then
			Expect(err).ToNot(BeNil())
			Expect(config).To(BeNil())
		})

		It("Test Capabilities call", func() {
			// Given
			// When
			config, err := config.NewConfig("")
			// Then
			Expect(err).To(BeNil())
			var addcaps, dropcaps []string
			caps, err := config.Capabilities("0", addcaps, dropcaps)
			Expect(err).To(BeNil())
			sort.Strings(caps)
			defaultCaps := config.Containers.DefaultCapabilities
			sort.Strings(defaultCaps)
			Expect(caps).To(BeEquivalentTo(defaultCaps))

			// Add all caps
			addcaps = []string{"all"}
			caps, err = config.Capabilities("root", addcaps, dropcaps)
			Expect(err).To(BeNil())
			sort.Strings(caps)
			Expect(caps).To(BeEquivalentTo(capabilities.AllCapabilities()))

			// Drop all caps
			dropcaps = []string{"all"}
			caps, err = config.Capabilities("", addcaps, dropcaps)
			Expect(err).To(BeNil())
			sort.Strings(caps)
			Expect(caps).ToNot(BeEquivalentTo([]string{}))

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
			Expect(err).To(BeNil())
			sort.Strings(caps)
			Expect(caps).To(BeEquivalentTo(expectedCaps))

			addcaps = []string{"NET_ADMIN", "cap_sys_admin"}
			dropcaps = []string{"FOWNER", "chown"}
			caps, err = config.Capabilities("", addcaps, dropcaps)
			Expect(err).To(BeNil())
			sort.Strings(caps)
			Expect(caps).To(BeEquivalentTo(expectedCaps))

			expectedCaps = []string{"CAP_NET_ADMIN", "CAP_SYS_ADMIN"}
			caps, err = config.Capabilities("notroot", addcaps, dropcaps)
			Expect(err).To(BeNil())
			sort.Strings(caps)
			Expect(caps).To(BeEquivalentTo(expectedCaps))
		})

		It("should succeed with default pull_policy", func() {
			err := sut.Engine.Validate()
			Expect(err).To(BeNil())
			Expect(sut.Engine.PullPolicy).To(Equal("missing"))

			sut.Engine.PullPolicy = config.DefaultPullPolicy
			err = sut.Engine.Validate()
			Expect(err).To(BeNil())
		})
		It("should fail with invalid pull_policy", func() {
			sut.Engine.PullPolicy = "invalidPullPolicy"
			err := sut.Engine.Validate()
			Expect(err).ToNot(BeNil())
		})
	})

})
