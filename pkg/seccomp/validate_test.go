// +build seccomp

package seccomp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProfile(t *testing.T) {
	for _, tc := range []struct {
		input     string
		shouldErr bool
	}{
		{ // success
			input:     `{"defaultAction": "SCMP_ACT_KILL"}`,
			shouldErr: false,
		},
		{ // Unmarshal failed
			input:     "wrong",
			shouldErr: true,
		},
		{ // setupSeccomp failed
			input:     `{"defaultAction": "SCMP_ACT_KILL", "architectures": ["SCMP_ARCH_X86"], "archMap": [{"architecture": "SCMP_ARCH_X86"}]}`,
			shouldErr: true,
		},
		{ // BuildFilter failed
			input:     `{"defaultAction": "SCMP_ACT_KILL", "architectures": ["SCMP_ARCH_X86"], "syscalls": [{ "name": "open" }]}`,
			shouldErr: true,
		},
	} {
		err := ValidateProfile(tc.input)
		if tc.shouldErr {
			require.NotNil(t, err)
		} else {
			require.Nil(t, err)
		}
	}
}
