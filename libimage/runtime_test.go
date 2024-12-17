//go:build !remote

package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/assert"
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

// Create a new Runtime that can be used for testing.
func testNewRuntime(t *testing.T, options ...testNewRuntimeOptions) *Runtime {
	workdir := t.TempDir()
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

	runtime, err := RuntimeFromStoreOptions(&RuntimeOptions{SystemContext: systemContext}, storeOptions)
	require.NoError(t, err)
	tmpd, err := tmpdir()
	require.NoError(t, err)
	require.Equal(t, runtime.systemContext.BigFilesTemporaryDir, tmpd)

	t.Cleanup(func() { _ = runtime.Shutdown(true) })

	sys := runtime.SystemContext()
	require.NotNil(t, sys)
	return runtime
}

func testRuntimePullImage(t *testing.T, r *Runtime, ctx context.Context, imageName string) {
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	_, err := r.Pull(ctx, imageName, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
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

func TestRuntimeListImagesAllImages(t *testing.T) {
	runtime := testNewRuntime(t)
	ctx := context.Background()

	// Prefetch alpine, busybox.
	testRuntimePullImage(t, runtime, ctx, "quay.io/libpod/alpine:latest")
	testRuntimePullImage(t, runtime, ctx, "quay.io/libpod/busybox:latest")

	images, err := runtime.ListImages(ctx, nil)
	require.NoError(t, err)

	require.Len(t, images, 2)
	var image_names []string
	for _, i := range images {
		image_names = append(image_names, i.Names()...)
	}
	assert.ElementsMatch(t,
		image_names,
		[]string{"quay.io/libpod/alpine:latest", "quay.io/libpod/busybox:latest"},
	)
}

func TestRuntimeListImagesByNames(t *testing.T) {
	runtime := testNewRuntime(t)
	ctx := context.Background()

	// Prefetch alpine, busybox.
	testRuntimePullImage(t, runtime, ctx, "quay.io/libpod/alpine:latest")
	testRuntimePullImage(t, runtime, ctx, "quay.io/libpod/busybox:latest")

	for _, test := range []struct {
		name     string
		fullName string
	}{
		{"alpine", "quay.io/libpod/alpine:latest"},
		{"busybox", "quay.io/libpod/busybox:latest"},
	} {
		images, err := runtime.ListImagesByNames([]string{test.name})
		require.NoError(t, err)
		require.Len(t, images, 1)
		require.Contains(t, images[0].Names(), test.fullName)
	}
	_, err := runtime.ListImagesByNames([]string{""})
	require.Error(t, err)
}
