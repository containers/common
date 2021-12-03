package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestFilterReference(t *testing.T) {
	busyboxLatest := "quay.io/libpod/busybox:latest"
	alpineLatest := "quay.io/libpod/alpine:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout

	pulledImages, err := runtime.Pull(ctx, busyboxLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	busybox := pulledImages[0]

	pulledImages, err = runtime.Pull(ctx, alpineLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	alpine := pulledImages[0]

	err = busybox.Tag("localhost/image:tag")
	require.NoError(t, err)
	err = alpine.Tag("localhost/image:another-tag")
	require.NoError(t, err)

	for _, test := range []struct {
		filter  string
		matches int
	}{
		{"image", 0},
		{"localhost/image", 2},
		{"localhost/image:tag", 1},
		{"localhost/image:another-tag", 1},
		{"localhost/*", 2},
		{"localhost/image:*tag", 2},
		{"busybox", 0},
		{"alpine", 0},
		{"quay.io/libpod/busybox", 1},
		{"quay.io/libpod/alpine", 1},
		{"quay.io/libpod", 0},
		{"quay.io/libpod/*", 2},
	} {
		listOptions := &ListImagesOptions{
			Filters: []string{"reference=" + test.filter},
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filter, listedImages)
	}
}
