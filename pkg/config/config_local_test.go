//go:build !remote
// +build !remote

package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/containers/common/libnetwork/types"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Config Local", func() {
	It("should not fail on invalid NetworkConfigDir", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		tmpfile := path.Join(os.TempDir(), "wrong-file")
		file, err := os.Create(tmpfile)
		gomega.Expect(err).To(gomega.BeNil())
		file.Close()
		defer os.Remove(tmpfile)
		defConf.Network.NetworkConfigDir = tmpfile
		defConf.Network.CNIPluginDirs.Set([]string{})

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should not fail on invalid CNIPluginDirs", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)

		// Given
		defConf.Network.NetworkConfigDir = validDirPath
		defConf.Network.CNIPluginDirs.Set([]string{invalidPath})

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should fail on invalid subnet pool", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		defConf.Network.NetworkConfigDir = validDirPath
		defConf.Network.CNIPluginDirs.Set([]string{validDirPath})

		net, _ := types.ParseCIDR("10.0.0.0/24")
		defConf.Network.DefaultSubnetPools = []SubnetPool{
			{Base: &net, Size: 16},
		}

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())

		defConf.Network.DefaultSubnetPools = []SubnetPool{
			{Base: &net, Size: 33},
		}

		// When
		err = defConf.Network.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("parse network subnet pool", func() {
		config, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		net1, _ := types.ParseCIDR("10.89.0.0/16")
		net2, _ := types.ParseCIDR("10.90.0.0/15")
		gomega.Expect(config.Network.DefaultSubnetPools).To(gomega.Equal(
			[]SubnetPool{{
				Base: &net1,
				Size: 24,
			}, {
				Base: &net2,
				Size: 24,
			}},
		))
	})

	It("parse dns port", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Network.DNSBindPort).To(gomega.Equal(uint16(0)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Network.DNSBindPort).To(gomega.Equal(uint16(1153)))
	})

	It("parse pasta_options", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Network.PastaOptions.Get()).To(gomega.HaveLen(0))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Network.PastaOptions.Get()).To(gomega.Equal([]string{"-t", "auto"}))
	})

	It("parse default_rootless_network_cmd", func() {
		// Given
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Network.DefaultRootlessNetworkCmd).To(gomega.Equal("slirp4netns"))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Network.DefaultRootlessNetworkCmd).To(gomega.Equal("pasta"))
	})

	It("should fail on invalid device mode", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Devices.Set([]string{"/dev/null:/dev/null:abc"})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid first device", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Devices.Set([]string{"wrong:/dev/null:rw"})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid second device", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Devices.Set([]string{"/dev/null:wrong:rw"})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid device", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Devices.Set([]string{invalidPath})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on wrong invalid device specification", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Devices.Set([]string{"::::"})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on bad timezone", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.TZ = "foo"

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should succeed on good timezone", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.TZ = "US/Eastern"

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on local timezone", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.TZ = "local"

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should fail on wrong DefaultUlimits", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.DefaultUlimits.Set([]string{invalidPath})

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should return containers engine env", func() {
		// Given
		expectedEnv := []string{"super=duper", "foo=bar"}
		// When
		config, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Env.Get()).To(gomega.BeEquivalentTo(expectedEnv))
		gomega.Expect(os.Getenv("super")).To(gomega.BeEquivalentTo("duper"))
		gomega.Expect(os.Getenv("foo")).To(gomega.BeEquivalentTo("bar"))
	})

	It("Expect Remote to be False", func() {
		// Given
		// When
		config, err := New(nil)
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Remote).To(gomega.BeFalse())
	})

	It("verify getDefaultEnv", func() {
		envs := []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		}
		// Given we do
		oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
		os.Setenv("CONTAINERS_CONF", "/dev/null")

		// When
		config, err := Default()

		// Undo that
		if envSet {
			os.Setenv("CONTAINERS_CONF", oldContainersConf)
		} else {
			os.Unsetenv("CONTAINERS_CONF")
		}
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

	It("write", func() {
		tmpfile := "containers.conf.test"
		oldContainersConf, envSet := os.LookupEnv("CONTAINERS_CONF")
		os.Setenv("CONTAINERS_CONF", tmpfile)
		defer func() {
			if envSet {
				os.Setenv("CONTAINERS_CONF", oldContainersConf)
			} else {
				os.Unsetenv("CONTAINERS_CONF")
			}
		}()

		config, err := ReadCustomConfig()
		gomega.Expect(err).To(gomega.BeNil())
		config.Containers.Devices.Set([]string{
			"/dev/null:/dev/null:rw",
			"/dev/sdc/",
			"/dev/sdc:/dev/xvdc",
			"/dev/sdc:rm",
		})
		boolTrue := true
		config.Containers.Env.Set([]string{"A", "B", "C"})
		config.Containers.Env.Attributes.Append = &boolTrue

		err = config.Write()
		gomega.Expect(err).To(gomega.BeNil())

		fi, err := os.Stat(tmpfile)
		gomega.Expect(err).To(gomega.BeNil())
		perm := int(fi.Mode().Perm())
		// 436 decimal = 644 octal
		gomega.Expect(perm).To(gomega.Equal(420))
		defer os.Remove(tmpfile)

		writtenConfig, err := ReadCustomConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(writtenConfig.Containers.Devices).To(gomega.BeEquivalentTo(config.Containers.Devices))
		gomega.Expect(writtenConfig.Containers.Env).To(gomega.BeEquivalentTo(config.Containers.Env))
		gomega.Expect(writtenConfig.Containers.Env.Attributes.Append).To(gomega.BeEquivalentTo(&boolTrue))
	})
	It("Default Umask", func() {
		// Given
		// When
		config, err := New(nil)
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Containers.Umask).To(gomega.Equal("0022"))
	})
	It("Set Umask", func() {
		// Given
		// When
		config, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Containers.Umask).To(gomega.Equal("0002"))
	})
	It("Should fail on bad Umask", func() {
		defConf, err := defaultConfig()
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(defConf).NotTo(gomega.BeNil())

		// Given
		defConf.Containers.Umask = "88888"

		// When
		err = defConf.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("Set Machine Enabled", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.MachineEnabled).To(gomega.Equal(false))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.MachineEnabled).To(gomega.Equal(true))
	})

	It("default netns", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Containers.NetNS).To(gomega.Equal("private"))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Containers.NetNS).To(gomega.Equal("bridge"))
	})

	It("should have a default secret driver", func() {
		// Given
		path := ""
		// When
		config, err := NewConfig(path)
		gomega.Expect(err).To(gomega.BeNil())
		// Then
		gomega.Expect(config.Secrets.Driver).To(gomega.Equal("file"))
	})

	It("should be possible to override the secret driver and options", func() {
		// Given
		path := "testdata/containers_override.conf"
		// When
		config, err := NewConfig(path)
		gomega.Expect(err).To(gomega.BeNil())
		// Then
		gomega.Expect(config.Secrets.Driver).To(gomega.Equal("pass"))
		gomega.Expect(config.Secrets.Opts).To(gomega.Equal(map[string]string{
			"key":  "foo@bar",
			"root": "/srv/password-store",
		}))
	})

	It("Set machine image path", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.Image).To(gomega.Equal("testing"))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		path := "https://example.com/$OS/$ARCH/foobar.ami"
		gomega.Expect(config2.Machine.Image).To(gomega.Equal(path))
		val := fmt.Sprintf("https://example.com/%s/%s/foobar.ami", runtime.GOOS, runtime.GOARCH)
		gomega.Expect(config2.Machine.URI()).To(gomega.BeEquivalentTo(val))
	})

	It("CompatAPIEnforceDockerHub", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.CompatAPIEnforceDockerHub).To(gomega.Equal(true))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.CompatAPIEnforceDockerHub).To(gomega.Equal(false))
	})

	It("ComposeProviders", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.ComposeProviders.Get()).To(gomega.Equal(getDefaultComposeProviders())) // no hard-coding to work on all platforms
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.ComposeProviders.Get()).To(gomega.Equal([]string{"/some/thing/else", "/than/before"}))
	})

	It("AddCompression", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.AddCompression.Get()).To(gomega.HaveLen(0)) // no hard-coding to work on all platforms
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.AddCompression.Get()).To(gomega.Equal([]string{"zstd", "zstd:chunked"}))
	})

	It("ComposeWarningLogs", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.ComposeWarningLogs).To(gomega.Equal(true))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.ComposeWarningLogs).To(gomega.Equal(false))
	})

	It("Set machine disk", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.DiskSize).To(gomega.Equal(uint64(100)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Machine.DiskSize).To(gomega.Equal(uint64(20)))
	})
	It("Set machine CPUs", func() {
		// Given
		cpus := runtime.NumCPU() / 2
		if cpus == 0 {
			cpus = 1
		}

		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.CPUs).To(gomega.Equal(uint64(cpus)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Machine.CPUs).To(gomega.Equal(uint64(1)))
	})
	It("Set machine memory", func() {
		// Given
		config, err := New(nil)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.Memory).To(gomega.Equal(uint64(2048)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Machine.Memory).To(gomega.Equal(uint64(1024)))
	})
})
