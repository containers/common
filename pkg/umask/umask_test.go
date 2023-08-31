package umask_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/containers/common/pkg/umask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkdirAllIgnoreUmask(t *testing.T) {
	t.Parallel()
	const testMode = os.FileMode(0o744)

	for _, tc := range []struct {
		name    string
		prepare func() string
		assert  func(string, error)
	}{
		{
			name: "success",
			prepare: func() string {
				dir := t.TempDir()
				return filepath.Join(dir, "foo", "bar")
			},
			assert: func(dir string, err error) {
				assert.NoError(t, err)

				// Assert $TMPDIR/foo/bar
				assert.DirExists(t, dir)
				info, err := os.Stat(dir)
				assert.NoError(t, err)
				assert.Equal(t, testMode, info.Mode().Perm())

				// Assert $TMPDIR/foo
				dir = filepath.Dir(dir)
				assert.DirExists(t, dir)
				info, err = os.Stat(dir)
				assert.NoError(t, err)
				assert.Equal(t, testMode, info.Mode().Perm())
			},
		},
		{
			name:    "success no dir to create",
			prepare: os.TempDir,
			assert: func(dir string, err error) {
				assert.NoError(t, err)
			},
		},
	} {
		prepare := tc.prepare
		assert := tc.assert

		t.Run(tc.name, func(t *testing.T) {
			old := syscall.Umask(0o077)
			defer syscall.Umask(old)

			t.Parallel()
			dir := prepare()

			err := umask.MkdirAllIgnoreUmask(dir, testMode)
			assert(dir, err)
		})
	}
}

func TestWriteFileIgnoreUmask(t *testing.T) {
	t.Parallel()
	const testMode = os.FileMode(0o744)

	for _, tc := range []struct {
		name    string
		prepare func() string
		assert  func(string, error)
	}{
		{
			name: "success",
			prepare: func() string {
				dir := t.TempDir()
				return filepath.Join(dir, "test")
			},
			assert: func(path string, err error) {
				assert.NoError(t, err)

				// Assert $TMPDIR/test
				assert.FileExists(t, path)
				info, err := os.Stat(path)
				assert.NoError(t, err)
				assert.Equal(t, testMode, info.Mode().Perm())
			},
		},
		{
			name: "failure path does not exist",
			prepare: func() string {
				path := t.TempDir()
				require.NoError(t, os.RemoveAll(path))
				return filepath.Join(path, "foo")
			},
			assert: func(path string, err error) {
				assert.Error(t, err)
			},
		},
	} {
		prepare := tc.prepare
		assert := tc.assert

		t.Run(tc.name, func(t *testing.T) {
			old := syscall.Umask(0o077)
			defer syscall.Umask(old)

			t.Parallel()
			path := prepare()

			err := umask.WriteFileIgnoreUmask(path, []byte("test"), testMode)
			assert(path, err)
		})
	}
}
