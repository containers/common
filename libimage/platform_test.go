package libimage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToPlatformString(t *testing.T) {
	for _, test := range []struct {
		arch, os, variant, expected string
	}{
		{"a", "b", "", "b/a"},
		{"a", "b", "c", "b/a/c"},
		{"", "", "c", "//c"}, // callers are responsible for the input
	} {
		platform := toPlatformString(test.arch, test.os, test.variant)
		require.Equal(t, platform, test.expected)
	}
}
