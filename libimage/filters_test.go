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
	err = alpine.Tag("localhost/another-image:tag")
	require.NoError(t, err)
	err = alpine.Tag("docker.io/library/image:another-tag")
	require.NoError(t, err)

	for _, test := range []struct {
		filter  string
		matches int
	}{
		{"image", 2},
		{"*mage*", 2},
		{"image:*", 2},
		{"image:tag", 1},
		{"image:another-tag", 1},
		{"localhost/image", 1},
		{"localhost/image:tag", 1},
		{"library/image", 1},
		{"docker.io/library/image*", 1},
		{"docker.io/library/image:*", 1},
		{"docker.io/library/image:another-tag", 1},
		{"localhost/*", 2},
		{"localhost/image:*tag", 1},
		{"localhost/*mage:*ag", 2},
		{"quay.io/libpod/busybox", 1},
		{"quay.io/libpod/alpine", 1},
		{"quay.io/libpod", 0},
		{"quay.io/libpod/*", 2},
		{"busybox", 1},
		{"alpine", 1},
	} {
		listOptions := &ListImagesOptions{
			Filters: []string{"reference=" + test.filter},
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filter, listedImages)
	}
}

func TestFilterManifest(t *testing.T) {
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

	pulledImages, err = runtime.Pull(ctx, alpineLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	_, err = runtime.CreateManifestList("manifest-alpine")
	require.NoError(t, err)

	for _, test := range []struct {
		filters []string
		matches int
	}{
		{nil, 3},
		{[]string{"manifest=false"}, 2},
		{[]string{"manifest=true"}, 1},
		{[]string{"reference=busybox"}, 1},
		{[]string{"reference=*alpine"}, 2},
		{[]string{"manifest=true", "reference=*alpine"}, 1},
		{[]string{"manifest=false", "reference=*alpine"}, 1},
		{[]string{"manifest=true", "reference=busybox"}, 0},
		{[]string{"manifest=false", "reference=busybox"}, 1},
	} {
		listOptions := &ListImagesOptions{
			Filters: test.filters,
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filters, listedImages)
	}
}
