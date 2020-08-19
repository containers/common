package supported

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/containers/common/pkg/apparmor/internal/supported/supportedfakes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestIsSupported(t *testing.T) {
	for _, tc := range []struct {
		prepare   func(*supportedfakes.FakeVerifierImpl) func()
		shoulderr bool
	}{
		{ // success with binary in /sbin
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(true)

				file, err := ioutil.TempFile("", "")
				require.Nil(t, err)
				fileInfo, err := file.Stat()
				require.Nil(t, err)
				mock.OsStatReturns(fileInfo, nil)

				return func() {
					require.Nil(t, os.RemoveAll(file.Name()))
				}
			},
			shoulderr: false,
		},
		{ // success with binary in $PATH
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(true)
				mock.OsStatReturns(nil, errors.New(""))
				mock.ExecLookPathReturns("", nil)

				return func() {}
			},
			shoulderr: false,
		},
		{ // error binary not in /sbin or $PATH
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(true)
				mock.OsStatReturns(nil, errors.New(""))
				mock.ExecLookPathReturns("", errors.New(""))
				return func() {}
			},
			shoulderr: true,
		},
		{ // error runc AppAmor not enabled
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(false)
				return func() {}
			},
			shoulderr: true,
		},
		{ // error rootless
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(true)
				return func() {}
			},
			shoulderr: true,
		},
	} {
		// Given
		sut := NewAppArmorVerifier()
		mock := &supportedfakes.FakeVerifierImpl{}
		cleanup := tc.prepare(mock)
		defer cleanup()
		sut.impl = mock

		// When
		err := sut.IsSupported()

		// Then
		if tc.shoulderr {
			require.NotNil(t, err)
		} else {
			require.Nil(t, err)
		}
	}
}
