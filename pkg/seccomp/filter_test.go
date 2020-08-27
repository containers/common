// +build seccomp

package seccomp

import (
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	libseccomp "github.com/seccomp/libseccomp-golang"
	"github.com/stretchr/testify/require"
)

func TestBuildFilter(t *testing.T) {
	for _, tc := range []struct {
		given  func() *specs.LinuxSeccomp
		expect func(*libseccomp.ScmpFilter, error)
	}{
		{ // Default profile
			given: func() *specs.LinuxSeccomp {
				sut, err := GetDefaultProfile(nil)
				require.Nil(t, err)
				return sut
			},
			expect: func(filter *libseccomp.ScmpFilter, err error) {
				require.Nil(t, err)
				require.NotNil(t, filter)
			},
		},
		{ // Spec nil
			given: func() *specs.LinuxSeccomp {
				return nil
			},
			expect: func(filter *libseccomp.ScmpFilter, err error) {
				require.Equal(t, ErrSpecNil, err)
				require.Nil(t, filter)
			},
		},
		{ // Spec empty
			given: func() *specs.LinuxSeccomp {
				return &specs.LinuxSeccomp{}
			},
			expect: func(filter *libseccomp.ScmpFilter, err error) {
				require.Equal(t, ErrSpecEmpty, err)
				require.Nil(t, filter)
			},
		},
		{ // Spec to seccomp failed
			given: func() *specs.LinuxSeccomp {
				sut, err := GetDefaultProfile(nil)
				require.Nil(t, err)
				sut.Syscalls[0].Action = "wrong"
				return sut
			},
			expect: func(filter *libseccomp.ScmpFilter, err error) {
				require.NotNil(t, err)
				require.Nil(t, filter)
			},
		},
	} {
		tc.expect(BuildFilter(tc.given()))
	}
}
