package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Make sure that loading images does not leave any artifacts in TMPDIR
	// behind (containers/podman/issues/14287).
	tmpdir := t.TempDir()
	os.Setenv("TMPDIR", tmpdir)
	defer func() {
		dir, err := os.ReadDir(tmpdir)
		require.NoError(t, err)
		require.Len(t, dir, 0)
		os.Unsetenv("TMPDIR")
	}()

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()
	loadOptions := &LoadOptions{}
	loadOptions.Writer = os.Stdout

	for _, test := range []struct {
		input       string
		expectError bool
		numImages   int
		names       []string
	}{
		// DOCKER ARCHIVE
		{"testdata/docker-name-only.tar.xz", false, 1, []string{"localhost/pretty-empty:latest"}},
		{"testdata/docker-registry-name.tar.xz", false, 1, []string{"example.com/empty:latest"}},
		{"testdata/docker-two-names.tar.xz", false, 1, []string{"localhost/pretty-empty:latest", "example.com/empty:latest"}},
		{"testdata/docker-two-images.tar.xz", false, 2, []string{"example.com/empty:latest", "example.com/empty/but:different"}},
		{"testdata/docker-unnamed.tar.xz", false, 1, []string{"sha256:ec9293436c2e66da44edb9efb8d41f6b13baf62283ebe846468bc992d76d7951"}},
		{"testdata/buildkit-docker.tar", false, 1, []string{"github.com/buildkit/archive:docker"}},

		// OCI ARCHIVE
		{"testdata/oci-name-only.tar.gz", false, 1, []string{"localhost/pretty-empty:latest"}},
		{"testdata/oci-non-docker-name.tar.gz", true, 0, nil},
		{"testdata/oci-registry-name.tar.gz", false, 1, []string{"example.com/empty:latest"}},
		{"testdata/oci-unnamed.tar.gz", false, 1, []string{"sha256:5c8aca8137ac47e84c69ae93ce650ce967917cc001ba7aad5494073fac75b8b6"}},
		{"testdata/buildkit-oci.tar", false, 1, []string{"github.com/buildkit/archive:oci"}},
	} {
		loadedImages, err := runtime.Load(ctx, test.input, loadOptions)
		if test.expectError {
			require.Error(t, err, test.input)
			continue
		}
		require.NoError(t, err, test.input)
		require.Equal(t, test.names, loadedImages, test.input)

		// Make sure that all returned names exist as images in the
		// local containers storage.
		ids := []string{} // later used for image removal
		names := [][]string{}
		for _, name := range loadedImages {
			image, resolvedName, err := runtime.LookupImage(name, nil)
			require.NoError(t, err, test.input)
			require.Equal(t, name, resolvedName, test.input)
			ids = append(ids, image.ID())
			names = append(names, image.Names())
		}

		// Now remove the image.
		rmReports, rmErrors := runtime.RemoveImages(ctx, ids, &RemoveImagesOptions{Force: true})
		require.Len(t, rmErrors, 0)
		require.Len(t, rmReports, test.numImages)

		// Now inspect the removal reports.
		for i, report := range rmReports {
			require.Equal(t, ids[i], report.ID, test.input)
			require.Equal(t, names[i], report.Untagged, test.input)
		}
	}
}
