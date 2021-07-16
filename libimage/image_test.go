package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestImageFunctions(t *testing.T) {
	// Note: this will resolve pull from the GCR registry (see
	// testdata/registries.conf).
	busyboxLatest := "docker.io/library/busybox:latest"
	busyboxDigest := "docker.io/library/busybox@"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	// Pull busybox:latest, get its digest and then perform a digest pull.
	// It's effectively pulling the same image twice but we need the digest
	// for some of the tests below.
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pulledImages, err := runtime.Pull(ctx, busyboxLatest, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	origDigest := pulledImages[0].Digest()
	busyboxDigest += origDigest.String()

	pulledImages, err = runtime.Pull(ctx, busyboxDigest, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	image := pulledImages[0]

	// Note that the tests below are primarily meant to be smoke tests to
	// catch broad regressions early on.
	require.NoError(t, image.reload())

	// The names history is stored in reverse order in c/storage.
	require.Equal(t, []string{busyboxDigest, busyboxLatest}, image.Names(), "names do not match")
	require.Equal(t, []string{busyboxLatest, busyboxDigest}, image.NamesHistory(), "names history does not match")

	require.NotNil(t, image.StorageImage())

	// Just make sure that the ID has 64 characters.
	require.True(t, len(image.ID()) == 64, "ID should be 64 characters long")

	// Make sure that the image we pulled by digest is the same one we
	// pulled by tag.
	require.Equal(t, origDigest.String(), image.Digest().String(), "digests of pulled images should match")

	// NOTE: we're recording two digests. One for the image and one for the
	// manifest list we chose it from.
	digests := image.Digests()
	require.Len(t, digests, 2)
	require.Equal(t, origDigest.String(), digests[0].String(), "first recoreded digest should be the one of the image")

	// Below mostly smoke tests.
	require.False(t, image.IsReadOnly())
	isDangling, err := image.IsDangling(ctx)
	require.NoError(t, err)
	require.False(t, isDangling)

	isIntermediate, err := image.IsIntermediate(ctx)
	require.NoError(t, err)
	require.False(t, isIntermediate)

	labels, err := image.Labels(ctx)
	require.NoError(t, err)

	require.True(t, image.TopLayer() != "", "non-empty top layer expected")

	parent, err := image.Parent(ctx)
	require.NoError(t, err)
	require.Nil(t, parent)

	hasChildren, err := image.HasChildren(ctx)
	require.NoError(t, err)
	require.False(t, hasChildren)

	children, err := image.Children(ctx)
	require.NoError(t, err)
	require.Nil(t, children)

	containers, err := image.Containers()
	require.NoError(t, err)
	require.Nil(t, containers)

	// Since we have no containers here, we can only smoke test.
	require.NoError(t, image.removeContainers(nil))
	require.Error(t, image.removeContainers(func(_ string) error {
		return errors.New("TEST")
	}))

	// Two items since both names are "Named".
	namedRepoTags, err := image.NamedRepoTags()
	require.NoError(t, err)
	require.Len(t, namedRepoTags, 2)
	require.Equal(t, busyboxLatest, namedRepoTags[1].String(), "unexpected named repo tag")
	require.Equal(t, busyboxDigest, namedRepoTags[0].String(), "unexpected named repo tag")

	// One item since only one name is "Tagged".
	namedTaggedRepoTags, err := image.NamedTaggedRepoTags()
	require.NoError(t, err)
	require.Len(t, namedTaggedRepoTags, 1, "unexpected named tagged repo tag")
	require.Equal(t, busyboxLatest, namedTaggedRepoTags[0].String(), "unexpected named tagged repo tag")
	repoTags, err := image.RepoTags()
	require.NoError(t, err)
	require.Len(t, repoTags, 1, "unexpected named tagged repo tag")
	require.Equal(t, busyboxLatest, repoTags[0], "unexpected named tagged repo tag")

	repoDigests, err := image.RepoDigests()
	require.NoError(t, err)
	require.Len(t, repoDigests, 2, "unexpected repo digests")

	mountPoint, err := image.Mount(ctx, nil, "")
	require.NoError(t, err)
	require.True(t, mountPoint != "", "non-empty mount point expected")

	sameMountPoint, err := image.Mountpoint()
	require.NoError(t, err)
	require.Equal(t, mountPoint, sameMountPoint, "mount points shoud be equal")

	require.NoError(t, image.Unmount(false))
	require.NoError(t, image.Unmount(true))

	// Same image -> same digest
	remoteRef, err := alltransports.ParseImageName("docker://" + busyboxDigest)
	require.NoError(t, err)
	hasDifferentDigest, err := image.HasDifferentDigest(ctx, remoteRef)
	require.NoError(t, err)
	require.False(t, hasDifferentDigest, "image with same digest should have the same manifest (and hence digest)")

	// Different images -> different digests
	remoteRef, err = alltransports.ParseImageName("docker://docker.io/library/alpine:latest")
	require.NoError(t, err)
	hasDifferentDigest, err = image.HasDifferentDigest(ctx, remoteRef)
	require.NoError(t, err)
	require.True(t, hasDifferentDigest, "another image should have a different digest")

	rawManifest, _, err := image.Manifest(ctx)
	require.NoError(t, err)
	require.True(t, len(rawManifest) > 0)

	size, err := image.Size()
	require.NoError(t, err)
	require.True(t, size > 0)

	// Now compare the inspect data to what we expect.
	imageData, err := image.Inspect(ctx, true)
	require.NoError(t, err)
	require.Equal(t, image.ID(), imageData.ID, "inspect data should match")
	require.Equal(t, repoTags, imageData.RepoTags, "inspect data should match")
	require.Len(t, imageData.RepoDigests, 2, "inspect data should match")
	require.Equal(t, size, imageData.Size, "inspect data should match")
	require.Equal(t, image.Digest().String(), imageData.Digest.String(), "inspect data should match")
	require.Equal(t, labels, imageData.Labels, "inspect data should match")
	require.Equal(t, image.NamesHistory(), imageData.NamesHistory, "inspect data should match")
}
