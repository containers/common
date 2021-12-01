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
	busyboxLatest := "quay.io/libpod/busybox:latest"
	busyboxDigest := "quay.io/libpod/busybox@"

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
	rmOptions := &RemoveImagesOptions{
		RemoveContainerFunc: func(_ string) error {
			return errors.New("TEST")
		},
		Force: true,
	}
	require.Error(t, image.removeContainers(rmOptions))

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
	require.Equal(t, mountPoint, sameMountPoint, "mount points should be equal")

	require.NoError(t, image.Unmount(false))
	require.NoError(t, image.Unmount(true))

	// Same image -> same digest
	remoteRef, err := alltransports.ParseImageName("docker://" + busyboxDigest)
	require.NoError(t, err)
	hasDifferentDigest, err := image.HasDifferentDigest(ctx, remoteRef, nil)
	require.NoError(t, err)
	require.False(t, hasDifferentDigest, "image with same digest should have the same manifest (and hence digest)")

	// Different images -> different digests
	remoteRef, err = alltransports.ParseImageName("docker://quay.io/libpod/alpine:latest")
	require.NoError(t, err)
	hasDifferentDigest, err = image.HasDifferentDigest(ctx, remoteRef, nil)
	require.NoError(t, err)
	require.True(t, hasDifferentDigest, "another image should have a different digest")

	rawManifest, _, err := image.Manifest(ctx)
	require.NoError(t, err)
	require.True(t, len(rawManifest) > 0)

	size, err := image.Size()
	require.NoError(t, err)
	require.True(t, size > 0)

	// Now compare the inspect data to what we expect.
	imageData, err := image.Inspect(ctx, &InspectOptions{WithParent: true, WithSize: true})
	require.NoError(t, err)
	require.Equal(t, image.ID(), imageData.ID, "inspect data should match")
	require.Equal(t, repoTags, imageData.RepoTags, "inspect data should match")
	require.Len(t, imageData.RepoDigests, 2, "inspect data should match")
	require.Equal(t, size, imageData.Size, "inspect data should match")
	require.Equal(t, image.Digest().String(), imageData.Digest.String(), "inspect data should match")
	require.Equal(t, labels, imageData.Labels, "inspect data should match")
	require.Equal(t, image.NamesHistory(), imageData.NamesHistory, "inspect data should match")
}

func TestInspectHealthcheck(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	imageName := "quay.io/libpod/healthcheck:config-only"
	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pulledImages, err := runtime.Pull(ctx, imageName, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	image := pulledImages[0]

	// Now compare the inspect data to what we expect.
	imageData, err := image.Inspect(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, imageData.HealthCheck, "health check should be found in config")
	require.Equal(t, []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"}, imageData.HealthCheck.Test, "health check should be found in config")
}

func TestTag(t *testing.T) {
	// Note: this will resolve pull from the GCR registry (see
	// testdata/registries.conf).
	busyboxLatest := "quay.io/libpod/busybox:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pulledImages, err := runtime.Pull(ctx, busyboxLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	image := pulledImages[0]

	digest := "sha256:adab3844f497ab9171f070d4cae4114b5aec565ac772e2f2579405b78be67c96"

	// Tag
	for _, test := range []struct {
		tag         string
		resolvesTo  string
		expectError bool
	}{
		{"foo", "localhost/foo:latest", false},
		{"docker.io/foo", "docker.io/library/foo:latest", false},
		{"quay.io/bar/foo:tag", "quay.io/bar/foo:tag", false},
		{"registry.com/$invalid", "", true},
		{digest, "", true},
		{"foo@" + digest, "", true},
		{"quay.io/foo@" + digest, "", true},
		{"", "", true},
	} {
		err := image.Tag(test.tag)
		if test.expectError {
			require.Error(t, err, "tag should have failed: %v", test)
			continue
		}
		require.NoError(t, err, "tag should have succeeded: %v", test)

		_, resolvedName, err := runtime.LookupImage(test.tag, nil)
		require.NoError(t, err, "image should have resolved locally: %v", test)
		require.Equal(t, test.resolvesTo, resolvedName, "image should have resolved correctly: %v", test)
	}

	// Check for specific error.
	err = image.Tag("foo@" + digest)
	require.True(t, errors.Cause(err) == errTagDigest, "check for specific digest error")
}

func TestUntag(t *testing.T) {
	// Note: this will resolve pull from the GCR registry (see
	// testdata/registries.conf).
	busyboxLatest := "quay.io/libpod/busybox:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout
	pulledImages, err := runtime.Pull(ctx, busyboxLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	image := pulledImages[0]

	digest := "sha256:adab3844f497ab9171f070d4cae4114b5aec565ac772e2f2579405b78be67c96"

	// Untag
	for _, test := range []struct {
		tag         string
		untag       string
		expectError bool
	}{
		{"foo", "foo", false},
		{"foo", "foo:latest", false},
		{"foo", "localhost/foo", false},
		{"foo", "localhost/foo:latest", false},
		{"quay.io/image/foo", "quay.io/image/foo", false},
		{"foo", "doNotExist", true},
		{"foo", digest, true},
		//		{"foo", "foo@" + digest, false},
		//		{"foo", "localhost/foo@" + digest, false},
	} {
		err := image.Tag(test.tag)
		require.NoError(t, err, "tag should have succeeded: %v", test)

		err = image.Untag(test.untag)
		if test.expectError {
			require.Error(t, err, "untag should have failed: %v", test)
			continue
		}
		require.NoError(t, err, "untag should have succeedded: %v", test)
		_, resolvedName, err := runtime.LookupImage(test.tag, nil)
		require.Error(t, err, "image should not resolve after untag anymore (%s): %v", resolvedName, test)
	}

	// Check for specific error.
	err = image.Untag(digest)
	require.True(t, errors.Cause(err) == errUntagDigest, "check for specific digest error")
}
