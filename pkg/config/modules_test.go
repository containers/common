package config

import (
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/unshare"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	testBaseHome = "testdata/modules/home/.config/containers/containers.conf"
	testBaseEtc  = "testdata/modules/etc/containers/containers.conf"
	testBaseUsr  = "testdata/modules/usr/share/containers/containers.conf"
)

func testSetModulePaths() *paths {
	wd, err := os.Getwd()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return &paths{
		usr:  filepath.Join(wd, testBaseUsr),
		etc:  filepath.Join(wd, testBaseEtc),
		home: filepath.Join(wd, testBaseHome),
		uid:  1000,
	}
}

var _ = Describe("Config Modules", func() {
	It("module directories", func() {
		dirs, err := ModuleDirectories()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(dirs).NotTo(gomega.BeNil())

		if unshare.IsRootless() {
			gomega.Expect(dirs).To(gomega.HaveLen(3))
		} else {
			gomega.Expect(dirs).To(gomega.HaveLen(2))
		}
	})

	It("resolve modules", func() {
		// This test makes sure that the correct module is being
		// returned.
		paths := testSetModulePaths()

		rootlessDirs := moduleDirectories(paths)
		gomega.Expect(rootlessDirs).To(gomega.HaveLen(3))
		gomega.Expect(rootlessDirs[0]).To(gomega.ContainSubstring(testBaseHome))
		gomega.Expect(rootlessDirs[1]).To(gomega.ContainSubstring(testBaseEtc))
		gomega.Expect(rootlessDirs[2]).To(gomega.ContainSubstring(testBaseUsr))

		paths.uid = 0
		rootfulDirs := moduleDirectories(paths)
		gomega.Expect(rootfulDirs).To(gomega.HaveLen(2))
		gomega.Expect(rootfulDirs[0]).To(gomega.ContainSubstring(testBaseEtc))
		gomega.Expect(rootfulDirs[1]).To(gomega.ContainSubstring(testBaseUsr))

		for _, test := range []struct {
			input       string
			expectedDir string
			mustFail    bool
			rootless    bool
		}{
			// Rootless
			{"first.conf", testBaseHome, false, true},
			{"second.conf", testBaseHome, false, true},
			{"third.conf", testBaseHome, false, true},
			{"sub/first.conf", testBaseHome, false, true},

			// Root + Rootless
			{"fourth.conf", testBaseEtc, false, false},
			{"sub/etc-only.conf", testBaseEtc, false, false},
			{"fifth.conf", testBaseUsr, false, false},
			{"sub/share-only.conf", testBaseUsr, false, false},
			{"none.conf", "", true, false},
		} {
			dirs := rootfulDirs
			if test.rootless {
				dirs = rootlessDirs
			}
			result, err := resolveModule(test.input, dirs)
			if test.mustFail {
				gomega.Expect(err).To(gomega.HaveOccurred())
				continue
			}
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(result).To(gomega.HaveSuffix(filepath.Join(test.expectedDir+".modules", test.input)))
		}
	})

	It("new config with modules", func() {
		paths := testSetModulePaths()

		wd, err := os.Getwd()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		options := &Options{Modules: []string{"none.conf"}}
		_, err = newLocked(options, paths)
		gomega.Expect(err).To(gomega.HaveOccurred()) // must error out

		options = &Options{}
		c, err := newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(options.additionalConfigs).To(gomega.BeEmpty()) // no module is getting loaded!
		gomega.Expect(c).NotTo(gomega.BeNil())
		gomega.Expect(c.LoadedModules()).To(gomega.BeEmpty())

		options = &Options{Modules: []string{"fourth.conf"}}
		c, err = newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1))
		// Make sure the returned module path is absolute.
		gomega.Expect(c.LoadedModules()).To(gomega.Equal([]string{filepath.Join(wd, "testdata/modules/etc/containers/containers.conf.modules/fourth.conf")}))

		options = &Options{Modules: []string{"fourth.conf"}}
		c, err = newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1))

		options = &Options{Modules: []string{"fourth.conf", "sub/share-only.conf", "sub/etc-only.conf"}}
		c, err = newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(3)) // 3 modules are getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.Containers.Env.Get()).To(gomega.Equal([]string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "usr share only"}))
		gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("etc only conf"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(3))

		options = &Options{Modules: []string{"third.conf"}}
		c, err = newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("home third"))

		paths.uid = 0
		c, err = newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("etc third"))
	})

	It("new config with modules and env variables", func() {
		paths := testSetModulePaths()

		t := GinkgoT()
		t.Setenv(containersConfOverrideEnv, "testdata/modules/override.conf")

		// Also make sure that absolute paths are loaded as is.
		wd, err := os.Getwd()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		absConf := filepath.Join(wd, "testdata/modules/home/.config/containers/containers.conf.modules/second.conf")

		options := &Options{Modules: []string{"fourth.conf", "sub/share-only.conf", absConf}}
		c, err := newLocked(options, paths)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(4)) // 2 modules + abs path + override conf are getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.Containers.Env.Get()).To(gomega.Equal([]string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "usr share only", "override conf always wins"}))
		gomega.Expect(c.Containers.Volumes.Get()).To(gomega.Equal([]string{"volume four", "home second"}))
	})
})
