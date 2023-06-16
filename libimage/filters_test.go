package libimage

import (
	"context"
	"os"
	"testing"
	"time"

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
		{"alpine@" + alpine.Digest().String(), 1},
		{"alpine:latest@" + alpine.Digest().String(), 1},
		{"quay.io/libpod/alpine@" + alpine.Digest().String(), 1},
		{"quay.io/libpod/alpine:latest@" + alpine.Digest().String(), 1},
		// Make sure that tags are ignored
		{"alpine:ignoreme@" + alpine.Digest().String(), 1},
		{"alpine:123@" + alpine.Digest().String(), 1},
		{"quay.io/libpod/alpine:hurz@" + alpine.Digest().String(), 1},
		{"quay.io/libpod/alpine:456@" + alpine.Digest().String(), 1},
		// Make sure that repo and digest must match
		{"alpine:busyboxdigest@" + busybox.Digest().String(), 0},
		{"alpine:busyboxdigest@" + busybox.Digest().String(), 0},
		{"quay.io/libpod/alpine:busyboxdigest@" + busybox.Digest().String(), 0},
		{"quay.io/libpod/alpine:busyboxdigest@" + busybox.Digest().String(), 0},
	} {
		listOptions := &ListImagesOptions{
			Filters: []string{"reference=" + test.filter},
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filter, listedImages)
	}
}

func TestFilterDigest(t *testing.T) {
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

	for _, test := range []struct {
		filter  string
		matches int
		id      string
	}{
		{string(busybox.Digest()), 1, busybox.ID()},
		{string(alpine.Digest()), 1, alpine.ID()},
	} {
		listOptions := &ListImagesOptions{
			Filters: []string{"digest=" + test.filter},
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filter, listedImages)
		require.Equal(t, listedImages[0].ID(), test.id)
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
		{[]string{"manifest!=false"}, 1},
		{[]string{"manifest!=true"}, 2},
		{[]string{"reference!=busybox"}, 2},
		{[]string{"reference!=*alpine"}, 1},
		{[]string{"manifest!=true", "reference!=*alpine"}, 1},
		{[]string{"manifest!=false", "reference!=*alpine"}, 0},
		{[]string{"manifest!=true", "reference!=busybox"}, 1},
		{[]string{"manifest!=false", "reference!=busybox"}, 1},
	} {
		listOptions := &ListImagesOptions{
			Filters: test.filters,
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filters, listedImages)
	}
}

func TestFilterAfterSinceBeforeUntil(t *testing.T) {
	testLatest := "quay.io/libpod/alpine_labels:latest"
	busyboxLatest := "quay.io/libpod/busybox:latest"
	alpineLatest := "quay.io/libpod/alpine:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout

	pulledImages, err := runtime.Pull(ctx, testLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	pulledImages, err = runtime.Pull(ctx, busyboxLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	pulledImages, err = runtime.Pull(ctx, alpineLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	alpine := pulledImages[0]

	err = alpine.Tag("test:tag")
	require.NoError(t, err)

	now := time.Until(time.Now()).String()

	for _, test := range []struct {
		filters []string
		matches int
	}{
		{nil, 3},
		{[]string{"after=test:tag"}, 2},
		{[]string{"after!=test:tag"}, 1},
		{[]string{"since=test:tag"}, 2},
		{[]string{"since!=test:tag"}, 1},
		{[]string{"before=test:tag"}, 0},
		{[]string{"before!=test:tag"}, 3},
		{[]string{"until=" + now}, 3},
		{[]string{"until!=" + now}, 0},
	} {
		listOptions := &ListImagesOptions{
			Filters: test.filters,
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filters, listedImages)
	}
}

func TestFilterIdLabel(t *testing.T) {
	testLatest := "quay.io/libpod/alpine_labels:latest"
	busyboxLatest := "quay.io/libpod/busybox:latest"
	alpineLatest := "quay.io/libpod/alpine:latest"

	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	pullOptions := &PullOptions{}
	pullOptions.Writer = os.Stdout

	pulledImages, err := runtime.Pull(ctx, testLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	pulledImages, err = runtime.Pull(ctx, busyboxLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)

	pulledImages, err = runtime.Pull(ctx, alpineLatest, config.PullPolicyMissing, pullOptions)
	require.NoError(t, err)
	require.Len(t, pulledImages, 1)
	alpine := pulledImages[0]

	identity := alpine.ID()

	for _, test := range []struct {
		filters []string
		matches int
	}{
		{nil, 3},
		{[]string{"id=" + identity}, 1},
		{[]string{"id!=" + identity}, 2},
		{[]string{"label=PODMAN=/usr/bin/podman run -it --name NAME -e NAME=NAME -e IMAGE=IMAGE IMAGE echo podman"}, 1},
		{[]string{"label!=PODMAN=/usr/bin/podman run -it --name NAME -e NAME=NAME -e IMAGE=IMAGE IMAGE echo podman"}, 2},
	} {
		listOptions := &ListImagesOptions{
			Filters: test.filters,
		}
		listedImages, err := runtime.ListImages(ctx, nil, listOptions)
		require.NoError(t, err, "%v", test)
		require.Len(t, listedImages, test.matches, "%s -> %v", test.filters, listedImages)
	}
}
