package libimage

import (
	"context"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestCreateManifestList(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	list, err := runtime.CreateManifestList("mylist")
	require.NoError(t, err)
	require.NotNil(t, list)
	initialID := list.ID()

	list, err = runtime.LookupManifestList("mylist")
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Equal(t, initialID, list.ID())

	_, rmErrors := runtime.RemoveImages(ctx, []string{"mylist"}, nil)
	require.Nil(t, rmErrors)

	_, err = runtime.LookupManifestList("nosuchthing")
	require.Error(t, err)
	require.Equal(t, errors.Cause(err), storage.ErrImageUnknown)

	_, err = runtime.Pull(ctx, "busybox", config.PullPolicyMissing, nil)
	require.NoError(t, err)
	_, err = runtime.LookupManifestList("busybox")
	require.Error(t, err)
	require.Equal(t, errors.Cause(err), ErrNotAManifestList)
}

// Following test ensure that `Tag` tags the manifest list instead of resolved image.
// Both the tags should point to same image id
func TestCreateAndTagManifestList(t *testing.T) {

	tagName := "testlisttagged"
	listName := "testlist"
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	list, err := runtime.CreateManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	manifestListOpts := &ManifestListAddOptions{All: true}
	_, err = list.Add(ctx, "docker://busybox", manifestListOpts)
	require.NoError(t, err)

	list, err = runtime.LookupManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	lookupOptions := &LookupImageOptions{ManifestList: true}
	image, _, err := runtime.LookupImage(listName, lookupOptions)
	require.NoError(t, err)
	require.NotNil(t, image)
	err = image.Tag(tagName)
	require.NoError(t, err, "tag should have succeeded: %s", tagName)

	taggedImage, _, err := runtime.LookupImage(tagName, lookupOptions)
	require.NoError(t, err)
	require.NotNil(t, taggedImage)

	// Both origin list and newly tagged list should point to same image id
	require.Equal(t, image.ID(), taggedImage.ID())
}
