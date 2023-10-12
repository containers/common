package platform

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToPlatformString(t *testing.T) {
	for _, test := range []struct {
		os, arch, variant, expected string
	}{
		{"a", "b", "", "a/b"},
		{"a", "", "", fmt.Sprintf("a/%s", runtime.GOARCH)},
		{"", "b", "", fmt.Sprintf("%s/b", runtime.GOOS)},
		{"a", "b", "c", "a/b/c"},
		{"", "", "", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
		{"", "", "c", fmt.Sprintf("%s/%s/c", runtime.GOOS, runtime.GOARCH)},
	} {
		platform := ToString(test.os, test.arch, test.variant)
		require.Equal(t, test.expected, platform)
	}
}
