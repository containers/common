package config

import (
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/unshare"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	testBaseHome = "testdata/modules/home/.config"
	testBaseEtc  = "testdata/modules/etc"
	testBaseUsr  = "testdata/modules/usr/share"
)

func testSetModulePaths() (func(), error) {
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	oldEtc := moduleBaseEtc
	oldUsr := moduleBaseUsr

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if err := os.Setenv("XDG_CONFIG_HOME", filepath.Join(wd, testBaseHome)); err != nil {
		return nil, err
	}

	moduleBaseEtc = filepath.Join(wd, testBaseEtc)
	moduleBaseUsr = filepath.Join(wd, testBaseUsr)

	return func() {
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
		moduleBaseEtc = oldEtc
		moduleBaseUsr = oldUsr
	}, nil
}

var _ = Describe("Config Modules", func() {
	It("module directories", func() {
		dirs, err := ModuleDirectories()
		gomega.Expect(err).To(gomega.BeNil())
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
		cleanUp, err := testSetModulePaths()
		gomega.Expect(err).To(gomega.BeNil())
		defer cleanUp()

		dirs, err := ModuleDirectories()
		gomega.Expect(err).To(gomega.BeNil())

		if unshare.IsRootless() {
			gomega.Expect(dirs).To(gomega.HaveLen(3))
			gomega.Expect(dirs[0]).To(gomega.ContainSubstring(testBaseHome))
			gomega.Expect(dirs[1]).To(gomega.ContainSubstring(testBaseEtc))
			gomega.Expect(dirs[2]).To(gomega.ContainSubstring(testBaseUsr))
		} else {
			gomega.Expect(dirs).To(gomega.HaveLen(2))
			gomega.Expect(dirs[0]).To(gomega.ContainSubstring(testBaseEtc))
			gomega.Expect(dirs[1]).To(gomega.ContainSubstring(testBaseUsr))
		}

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
			if test.rootless && !unshare.IsRootless() {
				continue
			}
			result, err := resolveModule(test.input, dirs)
			if test.mustFail {
				gomega.Expect(err).NotTo(gomega.BeNil())
				continue
			}
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(result).To(gomega.HaveSuffix(filepath.Join(test.expectedDir, moduleSubdir, test.input)))
		}
	})

	It("new config with modules", func() {
		cleanUp, err := testSetModulePaths()
		gomega.Expect(err).To(gomega.BeNil())
		defer cleanUp()

		wd, err := os.Getwd()
		gomega.Expect(err).To(gomega.BeNil())

		options := &Options{Modules: []string{"none.conf"}}
		_, err = New(options)
		gomega.Expect(err).NotTo(gomega.BeNil()) // must error out

		options = &Options{}
		c, err := New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(0)) // no module is getting loaded!
		gomega.Expect(c).NotTo(gomega.BeNil())
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(0))

		options = &Options{Modules: []string{"fourth.conf"}}
		c, err = New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1))
		// Make sure the returned module path is absolute.
		gomega.Expect(c.LoadedModules()).To(gomega.Equal([]string{filepath.Join(wd, "testdata/modules/etc/containers/containers.conf.modules/fourth.conf")}))

		options = &Options{Modules: []string{"fourth.conf"}}
		c, err = New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1))

		options = &Options{Modules: []string{"fourth.conf", "sub/share-only.conf", "sub/etc-only.conf"}}
		c, err = New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(3)) // 3 modules are getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.Containers.Env.Get()).To(gomega.Equal([]string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "usr share only"}))
		gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("etc only conf"))
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(3))

		options = &Options{Modules: []string{"third.conf"}}
		c, err = New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(1)) // 1 module is getting loaded!
		gomega.Expect(c.LoadedModules()).To(gomega.HaveLen(1))
		if unshare.IsRootless() {
			gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("home third"))
		} else {
			gomega.Expect(c.Network.DefaultNetwork).To(gomega.Equal("etc third"))
		}
	})

	It("new config with modules and env variables", func() {
		cleanUp, err := testSetModulePaths()
		gomega.Expect(err).To(gomega.BeNil())
		defer cleanUp()

		oldOverride := os.Getenv(containersConfOverrideEnv)
		defer func() {
			os.Setenv(containersConfOverrideEnv, oldOverride)
		}()

		err = os.Setenv(containersConfOverrideEnv, "testdata/modules/override.conf")
		gomega.Expect(err).To(gomega.BeNil())

		// Also make sure that absolute paths are loaded as is.
		wd, err := os.Getwd()
		gomega.Expect(err).To(gomega.BeNil())
		absConf := filepath.Join(wd, "testdata/modules/home/.config/containers/containers.conf.modules/second.conf")

		options := &Options{Modules: []string{"fourth.conf", "sub/share-only.conf", absConf}}
		c, err := New(options)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(options.additionalConfigs).To(gomega.HaveLen(4)) // 2 modules + abs path + override conf are getting loaded!
		gomega.Expect(c.Containers.InitPath).To(gomega.Equal("etc four"))
		gomega.Expect(c.Containers.Env.Get()).To(gomega.Equal([]string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "usr share only", "override conf always wins"}))
		gomega.Expect(c.Containers.Volumes.Get()).To(gomega.Equal([]string{"volume four", "home second"}))
	})
})
