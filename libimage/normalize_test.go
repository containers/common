package libimage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeName(t *testing.T) {
	const digestSuffix = "@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	for _, c := range []struct{ input, expected string }{
		{"#", ""}, // Clearly invalid
		{"example.com/busybox", "example.com/busybox:latest"},                                            // Qualified name-only
		{"example.com/busybox:notlatest", "example.com/busybox:notlatest"},                               // Qualified name:tag
		{"example.com/busybox" + digestSuffix, "example.com/busybox" + digestSuffix},                     // Qualified name@digest; FIXME? Should we allow tagging with a digest at all?
		{"example.com/busybox:notlatest" + digestSuffix, "example.com/busybox:notlatest" + digestSuffix}, // Qualified name:tag@digest
		{"busybox:latest", "localhost/busybox:latest"},                                                   // Unqualified name-only
		{"localhost/busybox", "localhost/busybox:latest"},                                                // Qualified with localhost
		{"ns/busybox:latest", "localhost/ns/busybox:latest"},                                             // Unqualified with a dot-less namespace
		{"docker.io/busybox:latest", "docker.io/library/busybox:latest"},                                 // docker.io without /library/
	} {
		res, err := NormalizeName(c.input)
		if c.expected == "" {
			assert.Error(t, err, c.input)
		} else {
			require.NoError(t, err, c.input)
			assert.Equal(t, c.expected, res.String())
		}
	}
}
