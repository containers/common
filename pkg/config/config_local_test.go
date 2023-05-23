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
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail in validating invalid PluginDir", func() {
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
		gomega.Expect(err).ToNot(gomega.BeNil())
	})

	It("should fail on invalid CNIPluginDirs", func() {
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
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should fail on invalid subnet pool", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(validDirPath)
		// Given
		sut.Network.NetworkConfigDir = validDirPath
		sut.Network.CNIPluginDirs = []string{validDirPath}

		net, _ := types.ParseCIDR("10.0.0.0/24")
		sut.Network.DefaultSubnetPools = []SubnetPool{
			{Base: &net, Size: 16},
		}

		// When
		err = sut.Network.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())

		sut.Network.DefaultSubnetPools = []SubnetPool{
			{Base: &net, Size: 33},
		}

		// When
		err = sut.Network.Validate()

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
		config, err := NewConfig("")
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
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Network.PastaOptions).To(gomega.BeNil())
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Network.PastaOptions).To(gomega.Equal([]string{"-t", "auto"}))
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

	It("should fail during runtime", func() {
		validDirPath, err := os.MkdirTemp("", "config-empty")
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

	It("should fail on bad timezone", func() {
		// Given
		sut.Containers.TZ = "foo"

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("should succeed on good timezone", func() {
		// Given
		sut.Containers.TZ = "US/Eastern"

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should succeed on local timezone", func() {
		// Given
		sut.Containers.TZ = "local"

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).To(gomega.BeNil())
	})

	It("should fail on wrong DefaultUlimits", func() {
		// Given
		sut.Containers.DefaultUlimits = []string{invalidPath}

		// When
		err := sut.Containers.Validate()

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
		gomega.Expect(config.Engine.Env).To(gomega.BeEquivalentTo(expectedEnv))
		gomega.Expect(os.Getenv("super")).To(gomega.BeEquivalentTo("duper"))
		gomega.Expect(os.Getenv("foo")).To(gomega.BeEquivalentTo("bar"))
	})

	It("Expect Remote to be False", func() {
		// Given
		// When
		config, err := NewConfig("")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Remote).To(gomega.BeFalse())
	})

	It("verify getDefaultEnv", func() {
		envs := []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"TERM=xterm",
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
		config, err := ReadCustomConfig()
		gomega.Expect(err).To(gomega.BeNil())
		config.Containers.Devices = []string{
			"/dev/null:/dev/null:rw",
			"/dev/sdc/",
			"/dev/sdc:/dev/xvdc",
			"/dev/sdc:rm",
		}

		err = config.Write()
		// Undo that
		if envSet {
			os.Setenv("CONTAINERS_CONF", oldContainersConf)
		} else {
			os.Unsetenv("CONTAINERS_CONF")
		}
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		fi, err := os.Stat(tmpfile)
		gomega.Expect(err).To(gomega.BeNil())
		perm := int(fi.Mode().Perm())
		// 436 decimal = 644 octal
		gomega.Expect(perm).To(gomega.Equal(420))
		defer os.Remove(tmpfile)
	})
	It("Default Umask", func() {
		// Given
		// When
		config, err := NewConfig("")
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
		// Given
		sut.Containers.Umask = "88888"

		// When
		err := sut.Containers.Validate()

		// Then
		gomega.Expect(err).NotTo(gomega.BeNil())
	})

	It("Set Machine Enabled", func() {
		// Given
		config, err := NewConfig("")
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
		config, err := NewConfig("")
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
		config, err := NewConfig("")
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
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.CompatAPIEnforceDockerHub).To(gomega.Equal(true))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Engine.CompatAPIEnforceDockerHub).To(gomega.Equal(false))
	})

	It("Set machine disk", func() {
		// Given
		config, err := NewConfig("")
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
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.CPUs).To(gomega.Equal(uint64(1)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Machine.CPUs).To(gomega.Equal(uint64(2)))
	})
	It("Set machine memory", func() {
		// Given
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Machine.Memory).To(gomega.Equal(uint64(2048)))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Machine.Memory).To(gomega.Equal(uint64(1024)))
	})
})
