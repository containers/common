//go:build !remote
// +build !remote

package libimage

import (
	"context"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestHistory(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	name := "quay.io/libpod/alpine:3.10.2"
	pullOptions := &PullOptions{}
	pulledImages, err := runtime.Pull(ctx, name, config.PullPolicyAlways, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	history, err := pulledImages[0].History(ctx)
	require.NoError(t, err)
	require.Len(t, history, 2)

	require.Equal(t, []string{name}, history[0].Tags)
	require.Equal(t, history[1].Tags, []string{name})
	require.NotEqual(t, history[0].Size, 0)
	require.NotEqual(t, history[1].Size, 0)
}
