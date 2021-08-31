package libimage

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/stretchr/testify/require"
)

func TestCorruptedLayers(t *testing.T) {
	// Regression tests for https://bugzilla.redhat.com/show_bug.cgi?id=1966872.
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout

	imageName := "quay.io/libpod/alpine_nginx:latest"

	exists, err := runtime.Exists(imageName)
	require.NoError(t, err, "image does not exist yet")
	require.False(t, exists, "image does not exist yet")

	pulledImages, err := runtime.Pull(ctx, imageName, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	image := pulledImages[0]

	// Inpsecting a healthy image should work.
	_, err = image.Inspect(ctx, false)
	require.NoError(t, err, "inspecting healthy image should work")

	exists, err = runtime.Exists(imageName)
	require.NoError(t, err, "healthy image exists")
	require.True(t, exists, "healthy image exists")

	// Disk usage works.
	_, err = runtime.DiskUsage(ctx)
	require.NoError(t, err, "disk usage works on healthy image")

	// Now remove one layer from the layers.json index in the storage.  The
	// image will still be listed in the container storage but attempting
	// to use it will yield "layer not known" errors.
	indexPath := filepath.Join(runtime.store.GraphRoot(), "vfs-layers/layers.json")
	data, err := ioutil.ReadFile(indexPath)
	require.NoError(t, err, "loading layers.json")
	layers := []*storage.Layer{}
	err = json.Unmarshal(data, &layers)
	require.NoError(t, err, "unmarshaling layers.json")
	require.LessOrEqual(t, 1, len(layers), "at least one layer must be present")

	// Now write back the layers without the first layer!
	data, err = json.Marshal(layers[1:])
	require.NoError(t, err, "unmarshaling layers.json")
	err = ioutils.AtomicWriteFile(indexPath, data, 0600) // nolint
	require.NoError(t, err, "writing back layers.json")

	image.reload() // clear the cached data

	// Now inspecting the image must fail!
	_, err = image.Inspect(ctx, false)
	require.Error(t, err, "inspecting corrupted image should fail")

	err = image.isCorrupted(imageName)
	require.Error(t, err, "image is corrupted")

	exists, err = runtime.Exists(imageName)
	require.NoError(t, err, "corrupted image exists should not fail")
	require.False(t, exists, "corrupted image should not be marked to exist")

	// Disk usage does not work.
	_, err = runtime.DiskUsage(ctx)
	require.Error(t, err, "disk usage does not work on corrupted image")
	require.Contains(t, err.Error(), "exists in local storage but may be corrupted", "disk usage reports corrupted image")

	// Now make sure that pull will detect the corrupted image and repulls
	// if needed which will repair the data corruption.
	pulledImages, err = runtime.Pull(ctx, imageName, config.PullPolicyNewer, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	image = pulledImages[0]

	// Inspecting a repaired image should work.
	_, err = image.Inspect(ctx, false)
	require.NoError(t, err, "inspecting repaired image should work")
}
