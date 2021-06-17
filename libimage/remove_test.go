package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRemoveImages(t *testing.T) {
	// Note: this will resolve pull from the GCR registry (see
	// testdata/registries.conf).
	busyboxLatest := "docker.io/library/busybox:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pulledImages, err := runtime.Pull(ctx, busyboxLatest, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	err = pulledImages[0].Tag("foobar")
	require.NoError(t, err)

	// containers/podman/issues/10685 - force removal on image with
	// multiple tags will only untag but not remove all tags including the
	// image.
	rmReports, rmErrors := runtime.RemoveImages(ctx, []string{"foobar"}, &RemoveImagesOptions{Force: true})
	require.Nil(t, rmErrors)
	require.Len(t, rmReports, 1)
	require.Equal(t, pulledImages[0].ID(), rmReports[0].ID)
	require.False(t, rmReports[0].Removed)
	require.Equal(t, []string{"localhost/foobar:latest"}, rmReports[0].Untagged)

	// The busybox image is still present even if foobar was force removed.
	exists, err := runtime.Exists(busyboxLatest)
	require.NoError(t, err)
	require.True(t, exists)
}
