//go:build !remote
// +build !remote

package libimage

import (
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

type testNewRuntimeOptions struct {
	registriesConfPath string
}

// Create a new Runtime that can be used for testing.  The second return value
// is a clean-up function that should be called by users to make sure all
// temporary test data gets removed.
func testNewRuntime(t *testing.T, options ...testNewRuntimeOptions) (runtime *Runtime, cleanup func()) {
	workdir, err := os.MkdirTemp("", "testStorageRuntime")
	require.NoError(t, err)
	storeOptions := &storage.StoreOptions{
		RunRoot:         workdir,
		GraphRoot:       workdir,
		GraphDriverName: "vfs",
	}

	// Make sure that the tests do not use the host's registries.conf.
	systemContext := &types.SystemContext{
		SystemRegistriesConfPath:    "testdata/registries.conf",
		SystemRegistriesConfDirPath: "/dev/null",
	}

	if len(options) == 1 && options[0].registriesConfPath != "" {
		systemContext.SystemRegistriesConfPath = options[0].registriesConfPath
	}

	runtime, err = RuntimeFromStoreOptions(&RuntimeOptions{SystemContext: systemContext}, storeOptions)
	require.NoError(t, err)
	tmpd, err := tmpdir()
	require.NoError(t, err)
	require.Equal(t, runtime.systemContext.BigFilesTemporaryDir, tmpd)

	cleanup = func() {
		_ = runtime.Shutdown(true)
		_ = os.RemoveAll(workdir)
	}

	sys := runtime.SystemContext()
	require.NotNil(t, sys)
	return runtime, cleanup
}

func TestTmpdir(t *testing.T) {
	tmpStr := "TMPDIR"
	tmp, tmpSet := os.LookupEnv(tmpStr)

	confStr := "CONTAINERS_CONF"
	conf, confSet := os.LookupEnv(confStr)

	os.Setenv(confStr, "testdata/containers.conf")
	_, err := config.Reload()
	require.NoError(t, err)

	tmpd, err := tmpdir()
	require.NoError(t, err)
	require.Equal(t, "/tmp/from/containers.conf", tmpd)

	if confSet {
		os.Setenv(confStr, conf)
	} else {
		os.Unsetenv(confStr)
	}
	_, err = config.Reload()
	require.NoError(t, err)

	os.Unsetenv(tmpStr)
	tmpd, err = tmpdir()
	require.NoError(t, err)
	require.Equal(t, "/var/tmp", tmpd)

	os.Setenv(tmpStr, "/tmp/test")
	tmpd, err = tmpdir()
	require.NoError(t, err)
	require.Equal(t, "/tmp/test", tmpd)
	if tmpSet {
		os.Setenv(tmpStr, tmp)
	} else {
		os.Unsetenv(tmpStr)
	}
}
