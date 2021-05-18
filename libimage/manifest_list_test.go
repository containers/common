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
