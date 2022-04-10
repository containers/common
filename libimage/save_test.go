package libimage

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	// Prefetch alpine, busybox.
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	_, err := runtime.Pull(ctx, "docker.io/library/alpine:latest", config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	_, err = runtime.Pull(ctx, "docker.io/library/busybox:latest", config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)

	// Save the two images into a multi-image archive.  This way, we can
	// reload the images for each test.
	saveOptions := &SaveOptions{}
	saveOptions.Writer = os.Stdout
	imageCache, err := ioutil.TempFile("", "saveimagecache")
	require.NoError(t, err)
	imageCache.Close()
	defer os.Remove(imageCache.Name())
	err = runtime.Save(ctx, []string{"alpine", "busybox"}, "docker-archive", imageCache.Name(), saveOptions)
	require.NoError(t, err)

	loadOptions := &LoadOptions{}
	loadOptions.Writer = os.Stdout

	// The table tests are smoke tests to exercise the different code
	// paths.  More detailed tests follow below.
	for _, test := range []struct {
		names       []string
		tags        []string
		format      string
		isDir       bool
		expectError bool
	}{
		// No `names`
		{nil, nil, "", false, true},
		{[]string{}, nil, "", false, true},
		// Invalid/unsupported format
		{[]string{"something"}, nil, "", false, true},
		{[]string{"something"}, nil, "else", false, true},
		// oci
		{[]string{"busybox"}, nil, "oci-dir", true, false},
		{[]string{"busybox"}, nil, "oci-archive", false, false},
		// oci-archive doesn't support multi-image archives
		{[]string{"busybox", "alpine"}, nil, "oci-archive", false, true},
		// docker
		{[]string{"busybox"}, nil, "docker-archive", false, false},
		{[]string{"busybox"}, []string{"localhost/tag:1", "quay.io/repo/image:tag"}, "docker-archive", false, false},
		{[]string{"busybox"}, nil, "docker-dir", true, false},
		{[]string{"busybox", "alpine"}, nil, "docker-archive", false, false},
		// additional tags and multi-images conflict
		{[]string{"busybox", "alpine"}, []string{"tag"}, "docker-archive", false, true},
	} {
		// First clean up all images and load the cache.
		_, rmErrors := runtime.RemoveImages(ctx, nil, nil)
		require.Nil(t, rmErrors)
		_, err = runtime.Load(ctx, imageCache.Name(), loadOptions)
		require.NoError(t, err)

		tmp, err := ioutil.TempDir("", "libimagesavetest")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)
		if !test.isDir {
			tmp += "/archive.tar"
		}

		saveOptions.AdditionalTags = test.tags
		err = runtime.Save(ctx, test.names, test.format, tmp, saveOptions)
		if test.expectError {
			require.Error(t, err, "%v", test)
			continue
		}
		require.NoError(t, err, "%v", test)

		// Now remove all images again and attempt to load the
		// previously saved ones.
		_, rmErrors = runtime.RemoveImages(ctx, nil, nil)
		require.Nil(t, rmErrors)

		namesAndTags := append(test.names, test.tags...) //nolint:gocritic // ignore "appendAssign: append result not assigned to the same slice"
		loadedImages, err := runtime.Load(ctx, tmp, loadOptions)
		require.NoError(t, err)
		require.Len(t, loadedImages, len(namesAndTags))

		// Now make sure that all specified names (and tags) resolve to
		// an image the local containers storage.  Note that names are
		// only preserved in archives.
		if strings.HasSuffix(test.format, "-dir") {
			continue
		}
		_, err = runtime.ListImages(ctx, namesAndTags, nil)
		require.NoError(t, err, "%v", test)
	}
}
