// +build !remote

package config

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

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
		expectedEnv := []string{"http_proxy=internal.proxy.company.com", "foo=bar"}
		// When
		config, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Engine.Env).To(gomega.BeEquivalentTo(expectedEnv))
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
		config.Containers.Devices = []string{"/dev/null:/dev/null:rw",
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

	It("Rootless networking", func() {
		// Given
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Containers.RootlessNetworking).To(gomega.Equal("slirp4netns"))
		// When
		config2, err := NewConfig("testdata/containers_default.conf")
		// Then
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config2.Containers.RootlessNetworking).To(gomega.Equal("cni"))
	})

	It("default netns", func() {
		// Given
		config, err := NewConfig("")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(config.Containers.NetNS).To(gomega.Equal(""))
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
})
