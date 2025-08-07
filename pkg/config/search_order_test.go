package config

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("search order", func() {
	It("config file parsing order", func() {
		// High level idea, we define 12 file lookup paths. Then we simulate parsing
		// of that file tree for all three cases, uid 0 (root), uid 1000 (reads rootless
		// specific config files) and uid 500 (also reads rootless config files but not
		// the uid 1000 specific one)
		//
		// To know that each file is getting parsed we use 12 config values and 12 files,
		// each file in the order is populated with one less config value.
		// So the first value only exits in the /usr/ file and then the next file writes
		// one less config value but instead increments the values by one so we can
		// differentiate what value is in what file.

		configPaths := []string{
			"/usr/share/containers/containers.conf",
			"/etc/containers/containers.conf",
			"/etc/containers/containers.conf.d/1.conf",
			"/etc/containers/containers.conf.d/2.conf",
			"/etc/containers/containers.rootless.conf",
			"/etc/containers/containers.rootless.conf.d/3.conf",
			"/etc/containers/containers.rootless.conf.d/4.conf",
			"/etc/containers/containers.rootless.conf.d/1000/5.conf",
			"/etc/containers/containers.rootless.conf.d/1000/6.conf",
			"/home/.config/containers/containers.conf",
			"/home/.config/containers/containers.conf.d/7.conf",
			"/home/.config/containers/containers.conf.d/8.conf",
		}

		configFileValues := []string{
			`apparmor_profile = "apparmor%d"`,
			`base_hosts_file = "base%d"`,
			`cgroupns = "cgroupns%d"`,
			`cgroups = "cgroups%d"`,
			`host_containers_internal_ip = "host%d"`,
			`init_path = "init%d"`,
			`ipcns = "ipcns%d"`,
			`log_driver = "logdriver%d"`,
			`log_tag = "logtag%d"`,
			`netns = "netns%d"`,
			`pidns = "pidns%d"`,
			`seccomp_profile = "seccomp%d"`,
		}

		test := func(config *Config, uid int, testname string) {
			gomega.Expect(config.Containers.ApparmorProfile).To(gomega.Equal("apparmor1"), testname)
			gomega.Expect(config.Containers.BaseHostsFile).To(gomega.Equal("base2"), testname)
			gomega.Expect(config.Containers.CgroupNS).To(gomega.Equal("cgroupns3"), testname)
			gomega.Expect(config.Containers.Cgroups).To(gomega.Equal("cgroups4"), testname)

			// This is a bit confusing but I did not find a way to make it better readable.
			// Basically we have three cases here:
			//  - uid 0 (root): should not read containers.rootless.conf, values must stay at 4
			//  - uid != 1000: should read containers.rootless.conf but not
			//                 containers.rootless.conf.d/1000/*.conf, values for the 1000 files must be 7
			//  - uid == 1000: all files are read so each value should be incremented by 1
			hostIP := "host5"
			initPath := "init6"
			ipcns := "ipcns7"
			logDriver := "logdriver7"
			logTag := "logtag7"
			switch uid {
			case 0:
				hostIP = "host4"
				initPath = "init4"
				ipcns = "ipcns4"
				logDriver = "logdriver4"
				logTag = "logtag4"
			case 1000:
				logDriver = "logdriver8"
				logTag = "logtag9"
			}
			gomega.Expect(config.Containers.HostContainersInternalIP).To(gomega.Equal(hostIP), testname)
			gomega.Expect(config.Containers.InitPath).To(gomega.Equal(initPath), testname)
			gomega.Expect(config.Containers.IPCNS).To(gomega.Equal(ipcns), testname)
			gomega.Expect(config.Containers.LogDriver).To(gomega.Equal(logDriver), testname)
			gomega.Expect(config.Containers.LogTag).To(gomega.Equal(logTag), testname)

			gomega.Expect(config.Containers.NetNS).To(gomega.Equal("netns10"), testname)
			gomega.Expect(config.Containers.PidNS).To(gomega.Equal("pidns11"), testname)
			gomega.Expect(config.Containers.SeccompProfile).To(gomega.Equal("seccomp12"), testname)
		}

		// sanity check
		gomega.Expect(configFileValues).To(gomega.HaveLen(len(configPaths)))

		tmpdir := GinkgoT().TempDir()
		for i, path := range configPaths {
			path := filepath.Join(tmpdir, path)
			err := os.MkdirAll(filepath.Dir(path), 0o755)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			f, err := os.Create(path)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			_, err = f.WriteString("[containers]\n")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			for _, val := range configFileValues[i:] {
				_, err = fmt.Fprintf(f, val+"\n", i+1)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
			f.Close()
		}

		paths := &paths{
			usr:  filepath.Join(tmpdir, "/usr/share/containers/containers.conf"),
			etc:  filepath.Join(tmpdir, "/etc/containers/containers.conf"),
			home: filepath.Join(tmpdir, "/home/.config/containers/containers.conf"),
			uid:  1000,
		}

		config, err := newLocked(&Options{}, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		test(config, 1000, "lookup with uid 1000")

		paths.uid = 500
		config, err = newLocked(&Options{}, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		test(config, 500, "lookup with uid 500")

		paths.uid = 0
		config, err = newLocked(&Options{}, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		test(config, 0, "lookup with uid 0 (root)")
	})
})
