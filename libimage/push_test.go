package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	// Prefetch alpine.
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	_, err := runtime.Pull(ctx, "docker.io/library/alpine:latest", config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)

	pushOptions := &PushOptions{}
	pushOptions.Writer = os.Stdout

	workdir, err := os.MkdirTemp("", "libimagepush")
	require.NoError(t, err)
	defer os.RemoveAll(workdir)

	for _, test := range []struct {
		source      string
		destination string
		expectError bool
	}{
		{"alpine", "dir:" + workdir + "/dir", false},
		{"alpine", "oci:" + workdir + "/oci", false},
		{"alpine", "oci-archive:" + workdir + "/oci-archive", false},
		{"alpine", "docker-archive:" + workdir + "/docker-archive", false},
		{"alpine", "containers-storage:localhost/another:alpine", false},
	} {
		_, err := runtime.Push(ctx, test.source, test.destination, pushOptions)
		if test.expectError {
			require.Error(t, err, "%v", test)
			continue
		}
		require.NoError(t, err, "%v", test)
		pulledImages, err := runtime.Pull(ctx, test.destination, config.PullPolicyAlways, pullOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, pulledImages, 1, "%v", test)
	}

	// Now there should only be two images: alpine in Docker format and
	// alpine in OCI format.
	listedImages, err := runtime.ListImages(ctx, nil, nil)
	require.NoError(t, err, "error listing images")
	require.Len(t, listedImages, 2, "there should only be two images (alpine in Docke/OCI)")

	// And now remove all of them.
	rmReports, rmErrors := runtime.RemoveImages(ctx, nil, nil)
	require.Len(t, rmErrors, 0)
	require.Len(t, rmReports, 2)

	for i, image := range listedImages {
		require.Equal(t, image.ID(), rmReports[i].ID)
		require.True(t, rmReports[i].Removed)
	}
}

func TestPushOtherPlatform(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	// Prefetch alpine.
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pullOptions.Architecture = "arm64"
	pulledImages, err := runtime.Pull(ctx, "docker.io/library/alpine:latest", config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	data, err := pulledImages[0].Inspect(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, "arm64", data.Architecture)

	pushOptions := &PushOptions{}
	pushOptions.Writer = os.Stdout
	tmp, err := os.CreateTemp("", "")
	require.NoError(t, err)
	tmp.Close()
	defer os.Remove(tmp.Name())
	_, err = runtime.Push(ctx, "docker.io/library/alpine:latest", "docker-archive:"+tmp.Name(), pushOptions)
	require.NoError(t, err)
}
