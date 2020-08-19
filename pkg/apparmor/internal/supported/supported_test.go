package supported

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/containers/common/pkg/apparmor/internal/supported/supportedfakes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSingleton(t *testing.T) {
	// Create the singleton
	sut := NewAppArmorVerifier()
	mock := &supportedfakes.FakeVerifierImpl{}
	sut.impl = mock
	mock.OsStatReturns(nil, errors.New(""))

	// Retrieve the mocked path
	const testBinaryPath = "/some/test/path"
	mock.ExecLookPathReturns(testBinaryPath, nil)
	res, err := sut.FindAppArmorParserBinary()
	require.Nil(t, err)
	require.Equal(t, testBinaryPath, res)

	// Make the mock fail
	mock.ExecLookPathReturns("", errors.New(""))

	// Check if we still return the memoized result
	res, err = sut.FindAppArmorParserBinary()
	require.Nil(t, err)
	require.Equal(t, testBinaryPath, res)

	// A new singleton instance should return the same memoized result
	sutNew := NewAppArmorVerifier()
	res, err = sutNew.FindAppArmorParserBinary()
	require.Nil(t, err)
	require.Equal(t, testBinaryPath, res)
}

func TestApparmorVerifier(t *testing.T) {
	for _, tc := range []struct {
		description string
		prepare     func(*supportedfakes.FakeVerifierImpl) func()
		shoulderr   bool
	}{
		{
			description: "success with binary in /sbin",
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
		{
			description: "success with binary in $PATH",
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(true)
				mock.OsStatReturns(nil, errors.New(""))
				mock.ExecLookPathReturns("", nil)

				return func() {}
			},
			shoulderr: false,
		},
		{
			description: "error binary not in /sbin or $PATH",
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(true)
				mock.OsStatReturns(nil, errors.New(""))
				mock.ExecLookPathReturns("", errors.New(""))
				return func() {}
			},
			shoulderr: true,
		},
		{
			description: "error runc AppAmor not enabled",
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(false)
				mock.RuncIsEnabledReturns(false)
				return func() {}
			},
			shoulderr: true,
		},
		{
			description: "error rootless",
			prepare: func(mock *supportedfakes.FakeVerifierImpl) func() {
				mock.UnshareIsRootlessReturns(true)
				return func() {}
			},
			shoulderr: true,
		},
	} {
		// Given
		sut := &ApparmorVerifier{impl: &defaultVerifier{}}
		mock := &supportedfakes.FakeVerifierImpl{}
		cleanup := tc.prepare(mock)
		defer cleanup()
		sut.impl = mock

		// When
		err := sut.IsSupported()

		// Then
		if tc.shoulderr {
			require.NotNil(t, err, tc.description)
		} else {
			require.Nil(t, err, tc.description)
		}
	}
}
