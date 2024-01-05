package filedriver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreAndLookupSecretData(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	err = tstdriver.Store("unique_id", []byte("somedata"))
	require.NoError(t, err)

	secretData, err := tstdriver.Lookup("unique_id")
	require.NoError(t, err)
	require.Equal(t, secretData, []byte("somedata"))
}

func TestStoreDupID(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	err = tstdriver.Store("unique_id", []byte("somedata"))
	require.NoError(t, err)

	err = tstdriver.Store("unique_id", []byte("somedata"))
	require.Error(t, err)
}

func TestLookupBogus(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	_, err = tstdriver.Lookup("bogus")
	require.Error(t, err)
}

func TestDeleteSecretData(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	err = tstdriver.Store("unique_id", []byte("somedata"))
	require.NoError(t, err)
	err = tstdriver.Delete("unique_id")
	require.NoError(t, err)
	data, err := tstdriver.Lookup("unique_id")
	require.Error(t, err)
	require.Nil(t, data)
}

func TestDeleteSecretDataNotExist(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	err = tstdriver.Delete("bogus")
	require.Error(t, err)
}

func TestList(t *testing.T) {
	tstdriver, err := NewDriver(t.TempDir())
	require.NoError(t, err)

	err = tstdriver.Store("unique_id", []byte("somedata"))
	require.NoError(t, err)
	err = tstdriver.Store("unique_id2", []byte("moredata"))
	require.NoError(t, err)

	data, err := tstdriver.List()
	require.NoError(t, err)
	require.Len(t, data, 2)
}
