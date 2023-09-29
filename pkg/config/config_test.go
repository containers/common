package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/containers/common/pkg/apparmor"
	"github.com/containers/common/pkg/capabilities"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Config", func() {
	Describe("ValidateConfig", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, err := NewConfig("")

			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defaultConfig.Containers.ApparmorProfile).To(gomega.Equal(apparmor.Profile))
			gomega.Expect(defaultConfig.Containers.BaseHostsFile).To(gomega.Equal(""))
			gomega.Expect(defaultConfig.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(defaultConfig.Containers.ReadOnly).To(gomega.BeFalse())
			gomega.Expect(defaultConfig.Engine.ServiceTimeout).To(gomega.BeEquivalentTo(5))
			gomega.Expect(defaultConfig.Engine.CompressionFormat).To(gomega.BeEquivalentTo("gzip"))
			gomega.Expect(defaultConfig.Engine.CompressionLevel).To(gomega.BeNil())
			gomega.Expect(defaultConfig.NetNS()).To(gomega.BeEquivalentTo("private"))
			gomega.Expect(defaultConfig.IPCNS()).To(gomega.BeEquivalentTo("shareable"))
			gomega.Expect(defaultConfig.Engine.InfraImage).To(gomega.BeEquivalentTo(""))
			if runtime.GOOS == "freebsd" {
				gomega.Expect(defaultConfig.Engine.ImageVolumeMode).To(gomega.BeEquivalentTo("nullfs"))
			} else {
				gomega.Expect(defaultConfig.Engine.ImageVolumeMode).To(gomega.BeEquivalentTo("bind"))
			}
			gomega.Expect(defaultConfig.Engine.SSHConfig).To(gomega.ContainSubstring("/.ssh/config"))
			gomega.Expect(defaultConfig.Engine.EventsContainerCreateInspectData).To(gomega.BeFalse())
			gomega.Expect(defaultConfig.Engine.DBBackend).To(gomega.BeEquivalentTo(stringBoltDB))
			gomega.Expect(defaultConfig.Engine.PodmanshTimeout).To(gomega.BeEquivalentTo(30))
			gomega.Expect(defaultConfig.Engine.AddCompression).To(gomega.BeNil())

			dbBackend, err := defaultConfig.DBBackend()
			gomega.Expect(dbBackend).To(gomega.BeEquivalentTo(DBBackendBoltDB))
			gomega.Expect(err).To(gomega.BeNil())
			path, err := defaultConfig.ImageCopyTmpDir()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(path).To(gomega.BeEquivalentTo("/var/tmp"))
		})

		It("should succeed with devices", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			// Given
			defConf.Containers.Devices = []string{
				"/dev/null:/dev/null:rw",
				"/dev/sdc/",
				"/dev/sdc:/dev/xvdc",
				"/dev/sdc:rm",
			}

			// When
			err = defConf.Containers.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail wrong max log size", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			// Given
			defConf.Containers.LogSizeMax = 1

			// When
			err = defConf.Validate()

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("should succeed with valid shm size", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			// Given
			defConf.Containers.ShmSize = "1024"

			// When
			err = defConf.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())

			// Given
			defConf.Containers.ShmSize = "64m"

			// When
			err = defConf.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail wrong shm size", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			// Given
			defConf.Containers.ShmSize = "-2"

			// When
			err = defConf.Validate()

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("Check SELinux settings", func() {
			defaultConfig, _ := NewConfig("")
			// EnableLabeling should match whether or not SELinux is enabled on the host
			gomega.Expect(defaultConfig.Containers.EnableLabeling).To(gomega.Equal(selinux.GetEnabled()))
			gomega.Expect(defaultConfig.Containers.EnableLabeledUsers).To(gomega.BeFalse())
		})
	})

	Describe("ValidateNetworkConfig", func() {
		It("should succeed with default config", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			// Given
			// When
			err = defConf.Network.Validate()

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})
	})

	Describe("readStorageTmp", func() {
		It("test image_copy_tmp_dir='storage'", func() {
			// Reload from new configuration file
			testFile := "testdata/temp.conf"
			content := `[engine]
image_copy_tmp_dir="storage"`
			err := os.WriteFile(testFile, []byte(content), os.ModePerm)
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			defer os.Remove(testFile)

			config, _ := NewConfig(testFile)
			path, err := config.ImageCopyTmpDir()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(path).To(gomega.ContainSubstring("containers/storage/tmp"))
			// Given we do
			oldTMPDIR, set := os.LookupEnv("TMPDIR")
			os.Setenv("TMPDIR", "/var/tmp/foobar")
			path, err = config.ImageCopyTmpDir()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(path).To(gomega.BeEquivalentTo("/var/tmp/foobar"))
			if set {
				os.Setenv("TMPDIR", oldTMPDIR)
			} else {
				os.Unsetenv("TMPDIR")
			}
		})
	})

	Describe("readConfigFromFile", func() {
		It("should succeed with default config", func() {
			// Given
			// When
			defaultConfig, _ := defaultConfig()
			// prior to reading local config, shows hard coded defaults
			gomega.Expect(defaultConfig.Containers.HTTPProxy).To(gomega.Equal(true))

			err := readConfigFromFile("testdata/containers_default.conf", defaultConfig)

			crunWasm := "crun-wasm"
			PlatformToOCIRuntimeMap := map[string]string{
				"wasi/wasm":   crunWasm,
				"wasi/wasm32": crunWasm,
				"wasi/wasm64": crunWasm,
			}

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
				"runj": {
					"/usr/local/bin/runj",
				},
				"crun": {
					"/usr/bin/crun",
					"/usr/local/bin/crun",
				},
				"crun-wasm": {
					"/usr/bin/crun-wasm",
					"/usr/sbin/crun-wasm",
					"/usr/local/bin/crun-wasm",
					"/usr/local/sbin/crun-wasm",
					"/sbin/crun-wasm",
					"/bin/crun-wasm",
					"/run/current-system/sw/bin/crun-wasm",
				},
				"runsc": {
					"/usr/bin/runsc",
					"/usr/sbin/runsc",
					"/usr/local/bin/runsc",
					"/usr/local/sbin/runsc",
					"/bin/runsc",
					"/sbin/runsc",
					"/run/current-system/sw/bin/runsc",
				},
				"youki": {
					"/usr/local/bin/youki",
					"/usr/bin/youki",
					"/bin/youki",
					"/run/current-system/sw/bin/youki",
				},
				"krun": {
					"/usr/bin/krun",
					"/usr/local/bin/krun",
				},
				"ocijail": {
					"/usr/local/bin/ocijail",
				},
			}

			pluginDirs := []string{
				"/usr/libexec/cni",
				"/tmp",
			}

			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			mounts := []string{
				"type=glob,source=/tmp/test2*,ro=true",
				"type=bind,source=/etc/services,destination=/etc/services,ro",
			}

			volumes := []string{
				"$HOME:$HOME",
			}

			newVolumes := []string{
				os.ExpandEnv("$HOME:$HOME"),
			}

			helperDirs := []string{
				"/somepath",
			}

			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defaultConfig.Engine.CgroupManager).To(gomega.Equal("systemd"))
			gomega.Expect(defaultConfig.Containers.Env).To(gomega.BeEquivalentTo(envs))
			gomega.Expect(defaultConfig.Containers.Mounts).To(gomega.BeEquivalentTo(mounts))
			gomega.Expect(defaultConfig.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(defaultConfig.Network.CNIPluginDirs).To(gomega.Equal(pluginDirs))
			gomega.Expect(defaultConfig.Network.NetavarkPluginDirs).To(gomega.Equal([]string{"/usr/netavark"}))
			gomega.Expect(defaultConfig.Engine.NumLocks).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(defaultConfig.Engine.OCIRuntimes).To(gomega.Equal(OCIRuntimeMap))
			gomega.Expect(defaultConfig.Engine.PlatformToOCIRuntime).To(gomega.Equal(PlatformToOCIRuntimeMap))
			gomega.Expect(defaultConfig.Containers.HTTPProxy).To(gomega.Equal(false))
			gomega.Expect(defaultConfig.Engine.NetworkCmdOptions).To(gomega.BeNil())
			gomega.Expect(defaultConfig.Engine.HelperBinariesDir).To(gomega.Equal(helperDirs))
			gomega.Expect(defaultConfig.Engine.ServiceTimeout).To(gomega.BeEquivalentTo(300))
			gomega.Expect(defaultConfig.Engine.InfraImage).To(gomega.BeEquivalentTo("k8s.gcr.io/pause:3.4.1"))
			gomega.Expect(defaultConfig.Machine.Volumes).To(gomega.BeEquivalentTo(volumes))
			gomega.Expect(defaultConfig.Engine.PodmanshTimeout).To(gomega.BeEquivalentTo(300))
			newV, err := defaultConfig.MachineVolumes()
			if newVolumes[0] == ":" {
				// $HOME is not set
				gomega.Expect(err).To(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(newV).To(gomega.BeEquivalentTo(newVolumes))
			}
		})

		It("test GetDefaultEnvEx", func() {
			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}
			httpEnvs := append([]string{"HTTP_PROXY=1.2.3.4"}, envs...)
			oldProxy, proxyEnvSet := os.LookupEnv("HTTP_PROXY")
			os.Setenv("HTTP_PROXY", "1.2.3.4")
			oldFoo, fooEnvSet := os.LookupEnv("foo")
			os.Setenv("foo", "bar")

			defaultConfig, _ := defaultConfig()
			gomega.Expect(defaultConfig.GetDefaultEnvEx(false, false)).To(gomega.BeEquivalentTo(envs))
			gomega.Expect(defaultConfig.GetDefaultEnvEx(false, true)).To(gomega.BeEquivalentTo(httpEnvs))
			gomega.Expect(strings.Join(defaultConfig.GetDefaultEnvEx(true, true), ",")).To(gomega.ContainSubstring("HTTP_PROXY"))
			gomega.Expect(strings.Join(defaultConfig.GetDefaultEnvEx(true, true), ",")).To(gomega.ContainSubstring("foo"))
			// Undo that
			if proxyEnvSet {
				os.Setenv("HTTP_PROXY", oldProxy)
			} else {
				os.Unsetenv("HTTP_PROXY")
			}
			if fooEnvSet {
				os.Setenv("foo", oldFoo)
			} else {
				os.Unsetenv("foo")
			}
		})

		It("should succeed with commented out configuration", func() {
			// Given
			// When
			conf := Config{}
			err := readConfigFromFile("testdata/containers_comment.conf", &conf)

			// Then
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail when file does not exist", func() {
			// Given
			// When
			conf := Config{}
			err := readConfigFromFile("/invalid/file", &conf)

			// Then
			gomega.Expect(err).NotTo(gomega.BeNil())
		})

		It("should fail when toml decode fails", func() {
			// Given
			// When
			conf := Config{}
			err := readConfigFromFile("config.go", &conf)

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

			defCaps := []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_NET_BIND_SERVICE",
				"CAP_SETFCAP",
				"CAP_SETGID",
				"CAP_SETPCAP",
				"CAP_SETUID",
				"CAP_SYS_CHROOT",
			}

			envs := []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}

			// Given we do
			oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
			os.Setenv("CONTAINERS_CONF", "/dev/null")
			// When
			config, err := NewConfig("")
			// Undo that
			if envSet {
				os.Setenv("CONTAINERS_CONF", oldContainersConf)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal(apparmor.Profile))
			gomega.Expect(config.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(config.Containers.Env).To(gomega.BeEquivalentTo(envs))
			gomega.Expect(config.Containers.UserNS).To(gomega.BeEquivalentTo(""))
			gomega.Expect(config.Network.CNIPluginDirs).To(gomega.Equal(DefaultCNIPluginDirs))
			gomega.Expect(config.Network.NetavarkPluginDirs).To(gomega.Equal(DefaultNetavarkPluginDirs))
			gomega.Expect(config.Engine.NumLocks).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(config.Engine.OCIRuntimes["runc"]).To(gomega.Equal(OCIRuntimeMap["runc"]))
			gomega.Expect(config.Containers.CgroupConf).To(gomega.BeNil())

			caps, _ := config.Capabilities("", nil, nil)
			gomega.Expect(caps).Should(gomega.Equal(defCaps))

			if useSystemd() {
				gomega.Expect(config.Engine.CgroupManager).To(gomega.BeEquivalentTo("systemd"))
			} else {
				gomega.Expect(config.Engine.CgroupManager).To(gomega.BeEquivalentTo("cgroupfs"))
			}
			if useJournald() {
				gomega.Expect(config.Engine.EventsLogger).To(gomega.BeEquivalentTo("journald"))
				gomega.Expect(config.Containers.LogDriver).To(gomega.BeEquivalentTo("journald"))
			} else {
				gomega.Expect(config.Engine.EventsLogger).To(gomega.BeEquivalentTo("file"))
				gomega.Expect(config.Containers.LogDriver).To(gomega.BeEquivalentTo("k8s-file"))
			}
			gomega.Expect(config.Engine.EventsLogFilePath).To(gomega.BeEquivalentTo(""))
			gomega.Expect(uint64(config.Engine.EventsLogFileMaxSize)).To(gomega.Equal(DefaultEventsLogSizeMax))
			gomega.Expect(config.Engine.PodExitPolicy).To(gomega.Equal(PodExitPolicyContinue))
			gomega.Expect(config.Engine.KubeGenerateType).To(gomega.Equal("pod"))
		})

		It("should success with valid user file path", func() {
			// Given
			// When
			config, err := NewConfig("testdata/containers_default.conf")
			// Then
			cgroupConf := []string{
				"memory.high=1073741824",
			}
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("container-default"))
			gomega.Expect(config.Containers.PidsLimit).To(gomega.BeEquivalentTo(2048))
			gomega.Expect(config.Containers.BaseHostsFile).To(gomega.BeEquivalentTo("/etc/hosts2"))
			gomega.Expect(config.Containers.HostContainersInternalIP).To(gomega.BeEquivalentTo("1.2.3.4"))
			gomega.Expect(config.Engine.ImageVolumeMode).To(gomega.BeEquivalentTo("tmpfs"))
			gomega.Expect(config.Engine.SSHConfig).To(gomega.Equal("/foo/bar/.ssh/config"))

			gomega.Expect(config.Engine.DBBackend).To(gomega.Equal(stringSQLite))
			dbBackend, err := config.DBBackend()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(dbBackend).To(gomega.BeEquivalentTo(DBBackendSQLite))
			gomega.Expect(config.Containers.CgroupConf).To(gomega.Equal(cgroupConf))
			gomega.Expect(*config.Containers.OOMScoreAdj).To(gomega.Equal(int(750)))
			gomega.Expect(config.Engine.KubeGenerateType).To(gomega.Equal("pod"))
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

			crunWasm := "crun-wasm"
			PlatformToOCIRuntimeMap := map[string]string{
				"hello":       "world",
				"wasi/wasm":   crunWasm,
				"wasi/wasm32": crunWasm,
				"wasi/wasm64": crunWasm,
			}

			// Also test `ImagePlatformToRuntimes
			runtimes := config.Engine.ImagePlatformToRuntime("wasi", "wasm")
			gomega.Expect(runtimes).To(gomega.Equal(crunWasm))

			// Then
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(config).ToNot(gomega.BeNil())
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("overridden-default"))
			gomega.Expect(config.Containers.LogDriver).To(gomega.Equal("journald"))
			gomega.Expect(config.Containers.LogTag).To(gomega.Equal("{{.Name}}|{{.ID}}"))
			gomega.Expect(config.Containers.LogSizeMax).To(gomega.Equal(int64(100000)))
			gomega.Expect(config.Containers.ReadOnly).To(gomega.BeTrue())
			gomega.Expect(config.Containers.Privileged).To(gomega.BeTrue())
			gomega.Expect(config.Engine.ImageParallelCopies).To(gomega.Equal(uint(10)))
			gomega.Expect(config.Engine.PlatformToOCIRuntime).To(gomega.Equal(PlatformToOCIRuntimeMap))
			gomega.Expect(config.Engine.ImageDefaultFormat).To(gomega.Equal("v2s2"))
			gomega.Expect(config.Engine.CompressionFormat).To(gomega.BeEquivalentTo("zstd:chunked"))
			gomega.Expect(config.Engine.EventsLogFilePath).To(gomega.BeEquivalentTo("/tmp/events.log"))
			gomega.Expect(config.Engine.EventsContainerCreateInspectData).To(gomega.BeTrue())
			path, err := config.ImageCopyTmpDir()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(path).To(gomega.BeEquivalentTo("/tmp/foobar"))
			gomega.Expect(uint64(config.Engine.EventsLogFileMaxSize)).To(gomega.Equal(uint64(500)))
			gomega.Expect(config.Engine.PodExitPolicy).To(gomega.BeEquivalentTo(PodExitPolicyStop))
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
			if runtime.GOOS != "linux" {
				Skip(fmt.Sprintf("capabilities not supported on %s", runtime.GOOS))
			}
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
			boundingSet, err := capabilities.BoundingSet()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(caps).To(gomega.BeEquivalentTo(boundingSet))

			// Drop all caps
			dropcaps = []string{"all"}
			caps, err = config.Capabilities("", boundingSet, dropcaps)
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
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			err = defConf.Engine.Validate()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf.Engine.PullPolicy).To(gomega.Equal("missing"))

			defConf.Engine.PullPolicy = DefaultPullPolicy
			err = defConf.Engine.Validate()
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should succeed case-insensitive", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			defConf.Engine.PullPolicy = "NeVer"
			err = defConf.Engine.Validate()
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("should fail with invalid pull_policy", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			defConf.Engine.PullPolicy = "invalidPullPolicy"
			err = defConf.Engine.Validate()
			gomega.Expect(err).ToNot(gomega.BeNil())
		})

		It("should fail with invalid database_backend", func() {
			defConf, err := defaultConfig()
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(defConf).NotTo(gomega.BeNil())

			defConf.Engine.DBBackend = ""
			err = defConf.Engine.Validate()
			gomega.Expect(err).ToNot(gomega.BeNil())
		})
	})

	Describe("Service Destinations", func() {
		ConfPath := struct {
			Value string
			IsSet bool
		}{}

		BeforeEach(func() {
			ConfPath.Value, ConfPath.IsSet = os.LookupEnv("CONTAINERS_CONF")
			conf, _ := os.CreateTemp("", "containersconf")
			os.Setenv("CONTAINERS_CONF", conf.Name())
		})

		AfterEach(func() {
			os.Remove(os.Getenv("CONTAINERS_CONF"))
			if ConfPath.IsSet {
				os.Setenv("CONTAINERS_CONF", ConfPath.Value)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
		})

		It("succeed to set and read", func() {
			cfg, err := ReadCustomConfig()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			cfg.Engine.ActiveService = "QA"
			cfg.Engine.ServiceDestinations = map[string]Destination{
				"QA": {
					URI:      "https://qa/run/podman/podman.sock",
					Identity: "/.ssh/id_rsa",
				},
			}
			err = cfg.Write()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// test that we do not write zero values to the file
			path, err := customConfigFile()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			f, err := os.Open(path)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			data, err := io.ReadAll(f)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(string(data)).ShouldNot(gomega.ContainSubstring("cpus"))
			gomega.Expect(string(data)).ShouldNot(gomega.ContainSubstring("disk_size"))
			gomega.Expect(string(data)).ShouldNot(gomega.ContainSubstring("memory"))

			cfg, err = ReadCustomConfig()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Expect(cfg.Engine.ActiveService, "QA")
			gomega.Expect(cfg.Engine.ServiceDestinations["QA"].URI,
				"https://qa/run/podman/podman.sock")
			gomega.Expect(cfg.Engine.ServiceDestinations["QA"].Identity,
				"/.ssh/id_rsa")
		})

		It("test addConfigs", func() {
			tmpFilePath := func(dir, prefix string) string {
				file, err := os.CreateTemp(dir, prefix)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				conf := file.Name() + ".conf"

				err = os.Rename(file.Name(), conf)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				return conf
			}
			configs := []string{
				"test1",
				"test2",
			}
			newConfigs, err := addConfigs("/bogus/path", configs)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(newConfigs).To(gomega.Equal(configs))

			dir, err := os.MkdirTemp("", "configTest")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			defer os.RemoveAll(dir)
			file1 := tmpFilePath(dir, "b")
			file2 := tmpFilePath(dir, "a")
			file3 := tmpFilePath(dir, "2")
			file4 := tmpFilePath(dir, "1")
			// create a file in dir that is not a .conf to make sure
			// it does not show up in configs
			_, err = os.CreateTemp(dir, "notconf")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			subdir, err := os.MkdirTemp(dir, "")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create a file in subdir, to make sure it does not
			// show up in configs
			_, err = os.CreateTemp(subdir, "")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			newConfigs, err = addConfigs(dir, configs)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			testConfigs := append(configs, []string{file4, file3, file2, file1}...)
			gomega.Expect(newConfigs).To(gomega.Equal(testConfigs))
		})

		It("test config errors", func() {
			conf := Config{}
			content := bytes.NewBufferString("")
			logrus.SetOutput(content)
			logrus.SetLevel(logrus.DebugLevel)
			err := readConfigFromFile("testdata/containers_broken.conf", &conf)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(conf.Containers.NetNS).To(gomega.Equal("bridge"))
			gomega.Expect(conf.Containers.Umask).To(gomega.Equal("0002"))
			gomega.Expect(content).To(gomega.ContainSubstring("Failed to decode the keys [\\\"foo\\\" \\\"containers.image_default_transport\\\"] from \\\"testdata/containers_broken.conf\\\""))
			logrus.SetOutput(os.Stderr)
		})

		It("test default config errors", func() {
			conf := Config{}
			content := bytes.NewBufferString("")
			logrus.SetOutput(content)
			err := readConfigFromFile("containers.conf", &conf)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(content.String()).To(gomega.Equal(""))
			logrus.SetOutput(os.Stderr)
		})
	})

	Describe("Farms", func() {
		ConfPath := struct {
			Value string
			IsSet bool
		}{}

		BeforeEach(func() {
			ConfPath.Value, ConfPath.IsSet = os.LookupEnv("CONTAINERS_CONF")
			conf, _ := os.CreateTemp("", "containersconf")
			os.Setenv("CONTAINERS_CONF", conf.Name())
		})

		AfterEach(func() {
			os.Remove(os.Getenv("CONTAINERS_CONF"))
			if ConfPath.IsSet {
				os.Setenv("CONTAINERS_CONF", ConfPath.Value)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
		})

		It("succeed to set and read", func() {
			cfg, err := ReadCustomConfig()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			cfg.Engine.ActiveService = "QA"
			cfg.Engine.ServiceDestinations = map[string]Destination{
				"QA": {
					URI:      "https://qa/run/podman/podman.sock",
					Identity: "/.ssh/id_rsa",
				},
			}
			err = cfg.Write()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// test that connections were written correctly
			cfg, err = ReadCustomConfig()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(cfg.Engine.ActiveService, "QA")
			gomega.Expect(cfg.Engine.ServiceDestinations["QA"].URI,
				"https://qa/run/podman/podman.sock")
			gomega.Expect(cfg.Engine.ServiceDestinations["QA"].Identity,
				"/.ssh/id_rsa")

			// Create farm
			cfg.Farms.Default = "Farm-1"
			cfg.Farms.List = map[string][]string{
				"Farm-1": {"QA"},
			}
			err = cfg.Write()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			cfg, err = ReadCustomConfig()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Expect(cfg.Farms.Default, "Farm-1")
			gomega.Expect(cfg.Farms.List["Farm-1"],
				"QA")
		})
	})

	Describe("Reload", func() {
		It("test new config from reload", func() {
			// Default configuration
			defaultTestFile := "testdata/containers_default.conf"
			oldEnv, set := os.LookupEnv("CONTAINERS_CONF")
			os.Setenv("CONTAINERS_CONF", defaultTestFile)
			cfg, err := Default()
			gomega.Expect(err).To(gomega.BeNil())
			if set {
				os.Setenv("CONTAINERS_CONF", oldEnv)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}

			// Reload from new configuration file
			testFile := "testdata/temp.conf"
			content := `[containers]
env=["foo=bar"]`
			err = os.WriteFile(testFile, []byte(content), os.ModePerm)
			defer os.Remove(testFile)
			gomega.Expect(err).To(gomega.BeNil())
			oldEnv, set = os.LookupEnv("CONTAINERS_CONF")
			os.Setenv("CONTAINERS_CONF", testFile)
			_, err = Reload()
			gomega.Expect(err).To(gomega.BeNil())
			newCfg, err := Default()
			gomega.Expect(err).To(gomega.BeNil())
			if set {
				os.Setenv("CONTAINERS_CONF", oldEnv)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}

			expectOldEnv := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
			expectNewEnv := []string{"foo=bar"}
			gomega.Expect(cfg.Containers.Env).To(gomega.Equal(expectOldEnv))
			gomega.Expect(newCfg.Containers.Env).To(gomega.Equal(expectNewEnv))
			// Reload change back to default global configuration
			_, err = Reload()
			gomega.Expect(err).To(gomega.BeNil())
		})
	})

	It("write default config should be empty", func() {
		defer os.Unsetenv("CONTAINERS_CONF")
		os.Setenv("CONTAINERS_CONF", "/dev/null")
		conf, err := ReadCustomConfig()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		f, err := os.CreateTemp("", "container-common-test")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		defer f.Close()
		defer os.Remove(f.Name())
		os.Setenv("CONTAINERS_CONF", f.Name())
		err = conf.Write()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		b, err := os.ReadFile(f.Name())
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		// config should only contain empty stanzas
		gomega.Expect(string(b)).To(gomega.
			Equal("[containers]\n\n[engine]\n\n[machine]\n\n[network]\n\n[secrets]\n\n[configmaps]\n\n[farms]\n"))
	})

	It("validate ImageVolumeMode", func() {
		for _, mode := range append(validImageVolumeModes, "") {
			err := ValidateImageVolumeMode(mode)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}
		err := ValidateImageVolumeMode("bogus")
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	It("CONTAINERS_CONF_OVERRIDE", func() {
		os.Setenv("CONTAINERS_CONF_OVERRIDE", "testdata/containers_override.conf")
		defer os.Unsetenv("CONTAINERS_CONF_OVERRIDE")
		config, err := NewConfig("")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("overridden-default"))

		// Make sure that _OVERRIDE is loaded even when CONTAINERS_CONF is set.
		os.Setenv("CONTAINERS_CONF", "testdata/containers_default.conf")
		defer os.Unsetenv("CONTAINERS_CONF")
		config, err = NewConfig("")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("overridden-default"))
		gomega.Expect(config.Containers.BaseHostsFile).To(gomega.Equal("/etc/hosts2"))
		gomega.Expect(config.Containers.EnableLabeledUsers).To(gomega.BeTrue())
	})
})
