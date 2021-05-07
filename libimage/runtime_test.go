package libimage

import (
	"io/ioutil"
	"os"
	"testing"

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

// Create a new Runtime that can be used for testing.  The second return value
// is a clean-up function that should be called by users to make sure all
// temporary test data gets removed.
func testNewRuntime(t *testing.T) (runtime *Runtime, cleanup func()) {
	workdir, err := ioutil.TempDir("", "testStorageRuntime")
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

	runtime, err = RuntimeFromStoreOptions(&RuntimeOptions{SystemContext: systemContext}, storeOptions)
	require.NoError(t, err)

	cleanup = func() {
		runtime.Shutdown(true)
		_ = os.RemoveAll(workdir)
	}
	return runtime, cleanup
}
