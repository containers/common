//go:build !remote
// +build !remote

package libimage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/assert"
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
	listOptions := ListImagesOptions{SetListData: true}
	listedImages, err := runtime.ListImages(ctx, nil, &listOptions)
	require.NoError(t, err, "error listing images")
	require.Len(t, listedImages, 2, "there should only be two images (alpine in Docke/OCI)")
	for _, image := range listedImages {
		require.NotNil(t, image.ListData.IsDangling, "IsDangling should be set")
	}

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

func TestPushWithForceCompression(t *testing.T) {
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

	// Push newly pulled alpine to directory with uncompressed blobs
	pushOptions := &PushOptions{}
	pushOptions.SystemContext = &types.SystemContext{}
	pushOptions.SystemContext.DirForceDecompress = true
	pushOptions.Writer = os.Stdout
	dirDest := t.TempDir()
	_, err = runtime.Push(ctx, "docker.io/library/alpine:latest", "dir:"+dirDest, pushOptions)
	require.NoError(t, err)

	// Pull uncompressed alpine from `dir:dirDest` as source.
	pullOptions = &PullOptions{}
	pullOptions.Writer = os.Stdout
	pullOptions.Architecture = "arm64"
	pulledImages, err = runtime.Pull(ctx, "dir:"+dirDest, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	// create `oci` image from uncompressed alpine.
	pushOptions = &PushOptions{}
	pushOptions.OciAcceptUncompressedLayers = true
	pushOptions.Writer = os.Stdout
	ociDest := t.TempDir()
	_, err = runtime.Push(ctx, "docker.io/library/alpine:latest", "oci:"+ociDest, pushOptions)
	require.NoError(t, err)

	// blobs from first push
	entries, err := os.ReadDir(filepath.Join(ociDest, "blobs", "sha256"))
	require.NoError(t, err)
	blobsFirstPush := []string{}
	for _, e := range entries {
		blobsFirstPush = append(blobsFirstPush, e.Name())
	}

	// Note: Compression is changed from `uncompressed` to `Gzip` but blobs
	// should still be same since `ForceCompressionFormat` is `false`.
	pushOptions = &PushOptions{}
	pushOptions.Writer = os.Stdout
	pushOptions.CompressionFormat = &compression.Gzip
	pushOptions.ForceCompressionFormat = false
	_, err = runtime.Push(ctx, "docker.io/library/alpine:latest", "oci:"+ociDest, pushOptions)
	require.NoError(t, err)

	// blobs from second push
	entries, err = os.ReadDir(filepath.Join(ociDest, "blobs", "sha256"))
	require.NoError(t, err)
	blobsSecondPush := []string{}
	for _, e := range entries {
		blobsSecondPush = append(blobsSecondPush, e.Name())
	}

	// All blobs of first push should be equivalent to blobs of
	// second push since same compression was used
	assert.Equal(t, blobsSecondPush, blobsFirstPush)

	// Note: Compression is changed from `uncompressed` to `Gzip` but blobs
	// should still be same since `ForceCompressionFormat` is `false`.
	pushOptions = &PushOptions{}
	pushOptions.Writer = os.Stdout
	pushOptions.CompressionFormat = &compression.Gzip
	pushOptions.ForceCompressionFormat = true
	_, err = runtime.Push(ctx, "docker.io/library/alpine:latest", "oci:"+ociDest, pushOptions)
	require.NoError(t, err)

	// collect blobs from third push
	entries, err = os.ReadDir(filepath.Join(ociDest, "blobs", "sha256"))
	require.NoError(t, err)
	blobsThirdPush := []string{}
	for _, e := range entries {
		blobsThirdPush = append(blobsThirdPush, e.Name())
	}

	// All blobs of third push should not be equivalent to blobs of
	// first push since new compression was used and `ForceCompressionFormat`
	// was `true`.
	assert.NotEqual(t, blobsThirdPush, blobsFirstPush)
}
